package build

var (
	// Version is the version number/tag assigned to this build
	Version = "dev-unknown"

	// FullCommit is the full git commit SHA for this build
	FullCommit = ""

	// Branch is the branch or tag name for this build.
	Branch = ""
)

// Info wraps the version/build  metadata
type Info struct {
	Version string
	Commit string
	Branch string
}