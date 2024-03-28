package dependencies

func DownloadBfg() error {
	err := downloadDependency("bfg", "jar",
		"https://repo1.maven.org/maven2/com/madgag/bfg/1.14.0/bfg-1.14.0.jar")
	return err
}
