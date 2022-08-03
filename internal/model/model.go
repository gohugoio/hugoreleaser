package model

type BuildContext struct {
	Project string `toml:"project"`
	Ref     string `toml:"ref"`
	Goos    string `toml:"goos"`
	Goarch  string `toml:"goarch"`
}
