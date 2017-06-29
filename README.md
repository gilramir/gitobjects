# gitobjects
Golang package for dealing with low-level aspects of Git

This is a work-in-progress

# NewRepo(directory)
Call this to get a new Repo object pointer. Pass in the workspace or .git directory path.

## StreamObjectsOfType(objectType)
This returns two channels which return object structs that are of one type, either
commit, blob, tree, or annotated-tag.

# Types
## Commit

## Tree
StreamBlobPathsUnique

## Blob
