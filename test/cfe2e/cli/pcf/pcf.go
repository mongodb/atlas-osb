package pcf

import "github.com/mongodb/atlas-osb/test/cfe2e/cli"

func Version() {
	session := cli.Execute("pcf", "version")
	session.Wait()
}

func Info() []byte {
	session := cli.ExecuteWithoutWriter("pcf", "cf-info")
	session.Wait("2m")
	return session.Out.Contents()
}
