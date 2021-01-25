package config

var (
	Version = true
	Debug = false
	Monitor = false
	SSHUsername = "user"
	SSHPassword = "password"
	SSHAddress = "6.6.6.x:22"
	FilePath = ""
	Reverse = false  //false: Push the local file to the remote end. true: Pull remote files to local
)
