package version

var (
	// Version is the current version of the oras.
	Version = "0.14.0-shizh.2"
	// BuildMetadata is the extra build time data
	BuildMetadata = "prototype"
	// GitCommit is the git sha1
	GitCommit = ""
	// GitTreeState is the state of the git tree
	GitTreeState = ""
)

// GetVersion returns the semver string of the version
func GetVersion() string {
	if BuildMetadata == "" {
		return Version
	}
	return Version + "+" + BuildMetadata
}
