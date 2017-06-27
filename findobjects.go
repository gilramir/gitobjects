package gitobjects

import (
	"bufio"
	"context"
	"fmt"
	"github.com/crewjam/errset"
	"github.com/pkg/errors"
	//	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Stream all Objects of a certain type on the Object channel. Once done reading the Object
// Channel, read the error channel to see if the stream stopped due to any error. The context
// can be used to cancel the request in progress.
func (self *GitRepo) StreamObjectsOfType(ctx context.Context, objectType string) (<-chan Object, <-chan error) {

	packFileChan := make(chan string)
	looseObjectSha1Chan := make(chan string)

	// Buffered so it can be written to at any time
	errorChan := make(chan error, 1)

	// One go-routine to find the pack files
	go _findPackFiles(ctx, self.gitDir, packFileChan)

	// One go-routine to find the loose object files
	go _findLooseObjectFiles(ctx, self.gitDir, looseObjectSha1Chan, errorChan)

	// Start n pack file processing go-routines
	numPackProcessors := 2
	packProcessorChans := make([]<-chan string, numPackProcessors)

	for i := 0; i < numPackProcessors; i++ {
		packObjectSha1Chan := make(chan string)
		packProcessorChans[i] = packObjectSha1Chan
		go _parsePackFile(ctx, self, packFileChan, packObjectSha1Chan, errorChan)
	}

	// Launch a go-routine to merge all the object sha1 chans into one
	objectSha1Chan := make(chan string)
	go _mergeObjectSha1Chans(looseObjectSha1Chan, packProcessorChans, objectSha1Chan)

	// Start n routines to process individual object sha1s
	numObjectProcessors := 4
	objectProcessorChans := make([]<-chan Object, numObjectProcessors)

	for i := 0; i < numObjectProcessors; i++ {
		objectProcessorChan := make(chan Object)
		objectProcessorChans[i] = objectProcessorChan
		go _parseObjectSha1(ctx, self, objectType, objectSha1Chan, objectProcessorChan, errorChan)
	}

	// Launch a go-routine to merge the object processor chans into one.
	// This is the routine that sends responsed ot the user
	responseChan := make(chan Object)
	go _mergeProcessedObjectChans(objectProcessorChans, responseChan, errorChan)

	return responseChan, errorChan
}

// Examine the .git directory, looking for pack files
func _findPackFiles(ctx context.Context, gitDir string, packFileChan chan<- string) {
	defer close(packFileChan)

	globPattern := filepath.Join(gitDir, "objects", "pack", "pack-*.idx")
	packFiles, err := filepath.Glob(globPattern)
	// The only error returned is for a bad pattern, which would be a programming mistake,
	// so panic.
	if err != nil {
		panic(err.Error())
	}

	for _, filename := range packFiles {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}
		packFileChan <- filename
	}
}

// Parse each pack file and send the object sha1s contained within it
func _parsePackFile(ctx context.Context, gitRepo *GitRepo, packFileChan <-chan string, packObjectSha1Chan chan<- string,
	errorChan chan<- error) {

	defer close(packObjectSha1Chan)

	for packFile := range packFileChan {
		packFileObject, err := os.Open(packFile)
		if err != nil {
			errorChan <- errors.Wrapf(err, "Opening pack file %s", packFile)
			return
		}

		cmd := gitRepo.Command([]string{"show-index"})
		cmd.Stdin = packFileObject
		cmd.Stdout = nil // During testing, I'm setting cmd.Stdout to os.Stdout, so this fixes is
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errorChan <- errors.Wrapf(err, "Getting stdout pipe for git show-index on pack file %s", packFile)
			_ = packFileObject.Close()
			return
		}
		err = cmd.Start()
		if err != nil {
			errorChan <- errors.Wrapf(err, "Starting git show-index on pack file %s", packFile)
			_ = packFileObject.Close()
			return
		}

		// Read the output line-by-line
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), " ")
			if len(fields) != 3 {
				errorChan <- errors.Errorf("Got unexpected line from show-index of %s: %s",
					packFile, scanner.Text())
				_ = packFileObject.Close()
				return
			}

			// The 2nd field has the sha1
			select {
			case <-ctx.Done():
				_ = packFileObject.Close()
				return
			default:
				break
			}
			packObjectSha1Chan <- fields[1]
		}

		errs := errset.ErrSet{}
		// Command error?
		cmdError := cmd.Wait()
		errs = append(errs, cmdError)

		// os.Close() error?
		fileError := packFileObject.Close()
		errs = append(errs, fileError)

		// Scanner error?
		scanError := scanner.Err()
		errs = append(errs, scanError)

		// Did any of those have an error?
		if errs.ReturnValue() != nil {
			errorChan <- errs.ReturnValue()
			return
		}
	}
}

// Find all loose object files
func _findLooseObjectFiles(ctx context.Context, gitDir string, looseObjectSha1Chan chan<- string, errorChan chan<- error) {
	defer close(looseObjectSha1Chan)

	sha1FilenameRegex, err := regexp.Compile(`^[0-9a-f]{38}$`)
	if err != nil {
		panic(err.Error())
	}

	sha1DirectoryRegex, err := regexp.Compile(`^[0-9a-f]{2}$`)
	if err != nil {
		panic(err.Error())
	}

	cancelSignalError := errors.New("stopped because of context cancel")

	err = filepath.Walk(filepath.Join(gitDir, "objects"),
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			base := filepath.Base(path)
			if sha1FilenameRegex.MatchString(base) {
				parentPath := filepath.Dir(path)
				parentBase := filepath.Base(parentPath)
				if sha1DirectoryRegex.MatchString(parentBase) {
					select {
					case <-ctx.Done():
						return cancelSignalError
					default:
						break
					}
					looseObjectSha1Chan <- parentBase + base
				}
			}
			return nil
		})

	if err != nil && err != cancelSignalError {
		errorChan <- err
		return
	}
}

// Take a sha1 and check the object type; if it is what we want, create an Object
// from it and send it.
func _parseObjectSha1(ctx context.Context, gitRepo *GitRepo, objectType string, objectSha1Chan <-chan string,
	objectProcessorChan chan<- Object, errorChan chan<- error) {

	defer close(objectProcessorChan)

	for sha1 := range objectSha1Chan {
		output, err := gitRepo.CmdOutput([]string{"cat-file", "-t", sha1})
		if err != nil {
			errorChan <- errors.Wrapf(err, "Getting object type for %s", sha1)
			return
		}
		type_ := strings.TrimRight(string(output), "\n")
		if type_ == objectType {
			select {
			case <-ctx.Done():
				return
			default:
				break
			}
			switch objectType {
			case "commit":
				objectProcessorChan <- &Commit{
					sha1: sha1,
				}
			default:
				panic(fmt.Sprintf("obj type %s not yet supported", objectType))
			}
		}
	}
}

// Merge the sha1s from 2 channels into 1 channel
func _mergeObjectSha1Chans(looseObjectSha1Chan <-chan string, packProcessorChans []<-chan string,
	objectSha1Chan chan<- string) {

	var wg sync.WaitGroup

	// Goroutine to shovel from one channel to another
	shovelFunc := func(c <-chan string) {
		for sha1 := range c {
			objectSha1Chan <- sha1
		}
		wg.Done()
	}
	go shovelFunc(looseObjectSha1Chan)
	wg.Add(1)
	for _, c := range packProcessorChans {
		go shovelFunc(c)
		wg.Add(1)
	}

	// Goroutine to close the output channel when the previous goroutines finish
	go func() {
		wg.Wait()
		close(objectSha1Chan)
	}()
}

// Merge the sha1s from the channels that have sha1s of the correct type,
// and send them down the responseChan back to the caller
func _mergeProcessedObjectChans(objectProcessorChans []<-chan Object, responseChan chan<- Object,
	errorChan chan<- error) {

	var wg sync.WaitGroup

	// Goroutine to shovel from one channel to another
	shovelFunc := func(c <-chan Object) {
		for obj := range c {
			responseChan <- obj
		}
		wg.Done()
	}
	for _, c := range objectProcessorChans {
		go shovelFunc(c)
		wg.Add(1)
	}

	// Goroutine to close the output channel when the previous goroutines finish
	go func() {
		wg.Wait()
		close(responseChan)
		// Also close the ErrorChan, as we're the last goroutine in the pipeline
		close(errorChan)
	}()
}
