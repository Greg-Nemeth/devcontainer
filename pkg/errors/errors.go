package errors

import "fmt"

type ContainerError struct {
	Description     string
	OriginalError   error
	ManageContainer bool
	ContainerId     string
	DockerParams    interface{}
	VolumeName      string
	RepositoryPath  string
	FolderPath      string
}

func (c *ContainerError) Error() string {
	if c.OriginalError != nil {
		return fmt.Sprintf("%s: %v", c.Description, c.OriginalError)
	}
	return c.Description
}
