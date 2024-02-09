package volume

type Volume struct {
	Alias         string
	Source        string
	MountType     string
	ContainerPath string
	MountArgs     string
}
