package main

import (
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"syncscp/config"
	"time"
)

var APPVersion = "1.0.0"

func SftpConnect(user, password, address string) (*sftp.Client, error) {
   auth := make([]ssh.AuthMethod, 0)
   auth = append(auth, ssh.Password(password))

   clientConfig := &ssh.ClientConfig{
      User:    user,
      Auth:    auth,
      Timeout: 30 * time.Second,
      HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
         return nil
      },
   }

   sshClient, err := ssh.Dial("tcp", address, clientConfig)
   if err != nil {
      log.Println("ERROR:", err)
      return nil, err
   }

   sftpClient, err := sftp.NewClient(sshClient)
   if err != nil {
      log.Println("ERROR:", err)
      return nil, err
   }

   return sftpClient, nil
}

func init() {
   flag.BoolVar(&config.Version, "v", false, "show version and exit")
   flag.BoolVar(&config.Debug, "d", false, "open debug mode")
   flag.BoolVar(&config.Monitor, "m", false, "monitoring mode. Only support monitor push the local \nfile to the remote end")

   flag.StringVar(&config.SSHAddress, "a", "", "Connect to address.")

   flag.StringVar(&config.SSHUsername, "u", "", "User for login.")
   flag.StringVar(&config.SSHPassword, "p", "", "Password to use when connecting to server. If password is \nnot given it's asked from the tty.")

   flag.StringVar(&config.FilePath, "f", "", "File path. Format is from/local.txt:to/remote.txt")
   flag.BoolVar(&config.Reverse, "r", false, "File transfer direction default: false \nfalse: Push the local file to the remote end. \ntrue: Pull remote files to local.")
}

func checkArgs() {
   if config.SSHAddress == "" {
      log.Fatalln("Please input address.")
   }
   if config.SSHUsername == "" {
      log.Fatalln("Please input user.")
   }
   if config.SSHPassword == "" {
      log.Fatalln("Please input password.")
   }
   if config.FilePath == "" {
      log.Fatalln("Please input file path")
   }
}

func getFilePath() (local string, remote string) {
   v := strings.Split(config.FilePath, ":")
   if len(v) != 2 {
      log.Fatalln("Please input the correct file path")
   }
   local = v[0]
   remote = v[1]
   return
}

func syncFile(localFilePath, remoteFilePath string) {
   client, err := SftpConnect(config.SSHUsername, config.SSHPassword, config.SSHAddress)
   if err != nil {
      log.Fatalln(err)
      return
   }
   defer func() {
      _ = client.Close()
   }()

   openFlag := os.O_RDONLY
   if config.Reverse == false {
   	  openFlag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
   }
   remoteFile, err := client.OpenFile(remoteFilePath, openFlag)
   if err != nil {
      log.Fatalln(err)
      return
   }
   defer func() {
      _ = remoteFile.Close()
   }()
   if config.Reverse == false {
      if err := remoteFile.Chmod(0644); err != nil {
         log.Fatalln(err)
         return
      }
   }

   openFlag = os.O_RDONLY
   if config.Reverse {
      openFlag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
   }
   localFile, err := os.OpenFile(localFilePath, openFlag, 0644)
   if err != nil {
      log.Fatalln(err)
      return
   }
   defer func() {
   	  _ = localFile.Close()
   }()

   var src io.ReadCloser = localFile
   var dst io.WriteCloser = remoteFile
   if config.Reverse {
      src = remoteFile
      dst = localFile
   }
   n, err := io.Copy(dst, src)
   if err != nil {
      log.Fatalln(err)
      return
   }
   log.Println("Transfer bytes:", n, "finished.")
}

func monitorFile(localFilePath, remoteFilePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(localFilePath)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				//syncFile(localFilePath, remoteFilePath)
				//log.Println("modified file:", event.Name, "OP:", event.Op)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func main() {
	flag.Parse()
	if config.Version {
		log.Println("Version:", APPVersion)
		return
	}
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	checkArgs()

	if config.Debug {
		log.SetFlags(log.Flags()|log.Lshortfile)
	} else {
		log.SetFlags(0)
	}

	localFilePath, remoteFilePath := getFilePath()

	if config.Monitor {
		if config.Reverse {
			log.Println("monitoring mode. Only support monitor push the local file to the remote end")
			return
		}
		monitorFile(localFilePath, remoteFilePath)
	} else {
		syncFile(localFilePath, remoteFilePath)
	}
}