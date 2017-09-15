package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

type Command struct {
	command string
	output  string
}

type Server struct {
	hostIp              string
	hostKeyFileLocation string
	hostUserName        string
	commands            []Command
}

func readConfig(configfileName, configFilepath string) (servers []Server) {
	var Servers []Server
	viper.SetConfigName("prop") // File extension is not required !!!
	viper.AddConfigPath("config")

	err := viper.ReadInConfig()
	if err != nil {
		panic("Problem Reading config File !!!")
	} else {
		numberOfHosts := viper.GetInt("counters.numberOfHosts")
		Servers = make([]Server, numberOfHosts)
		for i := 0; i < numberOfHosts; i++ {
			ipaddr := viper.GetString("ip.ip" + strconv.Itoa(i+1))
			username := viper.GetString("usernames.user" + strconv.Itoa(i+1))
			keyfile := viper.GetString("keyfiles.key" + strconv.Itoa(i+1))
			commandList := strings.Split(viper.GetString("server_commands.s"+strconv.Itoa(i+1)), ";")
			serverCommands := make([]Command, len(commandList))
			for j := 0; j < len(commandList); j++ {
				serverCommands[j] = Command{command: viper.GetString("all_commands." + commandList[j]), output: ""}
			}
			Servers[i] = Server{hostIp: ipaddr, hostUserName: username, hostKeyFileLocation: keyfile, commands: serverCommands}

		}

	}
	return Servers
}
func getKeyFile(location string) (key ssh.Signer, err error) {
	file := location
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return
	}
	return
}

func executeCommands(s *Server, wg *sync.WaitGroup) {

	//Get the Key
	key, err := getKeyFile(s.hostKeyFileLocation)
	if err != nil {
		panic(err)
	}

	//This is the bare minimum config
	config := &ssh.ClientConfig{
		User: s.hostUserName,
		//This is BAD!!!! I am using it for testing. Dont do it for prod
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	client, err := ssh.Dial("tcp", s.hostIp, config)
	if err != nil {
		log.Fatal("Error in Dialing: ", err)
	}
	//Can interact with multiple sessions for each command
	for k := 0; k < len(s.commands); k++ {

		session, err := client.NewSession()
		if err != nil {
			log.Fatal("Failed to create session: ", err)
		}
		defer session.Close()
		var b bytes.Buffer
		session.Stdout = &b
		if err := session.Run(s.commands[k].command); err != nil {
			log.Fatal("Failed to run: " + err.Error())
		}
		fmt.Println("Server: ", s.hostIp, ", Command: ", s.commands[k].command, " output: ", b.String())
	}
	defer wg.Done()
}

func main() {
	serverList := readConfig("prop", "config")
	//Can potentially use go routines here for concurrent execution of servers
	//Note: Commands in the same server will run sequentially
	var wg sync.WaitGroup

	for i := 0; i < len(serverList); i++ {
		//Similar to java thread joining
		wg.Add(1)
		//Starts a lighweight thread.
		go executeCommands(&serverList[i], &wg)
	}
	wg.Wait()
}
