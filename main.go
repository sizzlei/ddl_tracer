package main


import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"ddl_tracer/lib"
	"flag"
	"strings"
	"os"
	"github.com/loyalid/slack-incoming-webhook-go"
)

const (
	tracer = "DDL_Tracer"
	version = "0.1.0"
	author = "DBA"
	slackFormat = "Server: *%s* (%s)\nSchema : *%s* \nTarget : *%s*\n*Information* :\n%s \n"
	initSlackFormat ="Server: *%s*\nStatus : *%s*\nDescription : *%s*\n"
)

var Config *lib.Conf

func main(){
	var opt,pass,conf string
	flag.StringVar(&opt,"opt","","Tracer Exe Option")
	flag.StringVar(&pass,"password","","Encrypt Pass")
	flag.StringVar(&conf,"conf","config.yml","tracer Configure")

	flag.Parse()

	log.Infof("Regist tracer : %s Version : %s",tracer,version)

	// Option Check
	var encPass string
	if strings.ToUpper(opt) == "ENCRYPT" {
		encPass = lib.Crypto(1,pass)
		log.Infof("Password Encrypt Complete : %s",encPass)
		os.Exit(1)
	}

	// Configure Read
	config, err := lib.ConfReader(conf)
	if err != nil {
		log.Errorf("Config Setup Fail : %s", err)
		os.Exit(1)
	}

	Config = &config
	log.Infof("Config Load Success.")


	// Check Logic
	switch strings.ToUpper(opt) {
	case "INIT":
		log.Infof("Set Option %s.",opt)

		// Loop Server
		for _, servers := range Config.Servers {
			log.Infof("Set Initialize %s.",servers.Host)
			
			// Path
			hostPath := fmt.Sprintf("%s/%s",Config.Tracer.Datapath,servers.Alias)
			log.Infof("Setup Host Path : %s",hostPath)

			// Make Path
			_ = os.RemoveAll(hostPath)
			err := lib.MakePath(hostPath)
			if err != nil {
				log.Errorf("Failed to Setup Path. %s",err)
			}
			servers.HostPath = hostPath

			// Initialize Start
			err = lib.SetInit(servers,Config)
			if err != nil {
				log.Errorf("Failed to %s Initialize Process. %s",servers.Host,err)
			} 
			
			// Init Complete Send Msg
			var msg string
			if err != nil {
				msg = fmt.Sprintf(initSlackFormat,version,author,servers.Alias,"Errors",err)
			} else {
				msg = fmt.Sprintf(initSlackFormat,servers.Alias,"Complete","Definition Initialize Complete.")
			}
			slack.PostMessage(Config.Tracer.Webhook, msg)
		}

	case "CHECK":
		log.Infof("Set Option %s.",opt)

		// Loop Server
		for _, servers := range Config.Servers {
			log.Infof("Check Definition %s.",servers.Host)

			// Path
			hostPath := fmt.Sprintf("%s/%s",Config.Tracer.Datapath,servers.Alias)
			servers.HostPath = hostPath

			// Loop DB
			for _, db := range servers.Db {
				// DB Table column file load
				fileMap, err := lib.GetColumns(servers,db)
				if err != nil {
					log.Errorf("Failed to Column file load %s from %s.",db,servers.Host)
				}

				dbMap, err := lib.GetDefinition(servers, db, Config)
				if err != nil {
					log.Errorf("Failed to Information Schema Load %s from %s.",db,servers.Host)
				}

				// Definistion Diff Check
				diffData := lib.DiffMap(hostPath,db,fileMap,dbMap)

				

				// Diff Slice
				for i, d := range diffData {
					switch (d.Div) {
					case "table":
						switch (d.Result) {
						case "added":
							// Get Table ddl
							ddl, err := lib.GetTablesDDL(servers, Config, d)
							if err != nil {
								log.Errorf("Failed to Get DDL %s from %s",d.Table,d.Db)
							}
							diffData[i].Ddl = ddl
						case "dropped":
							diffData[i].Ddl = fmt.Sprintf("DROP Table %s.%s",d.Db,d.Table)
						}
					case "column":
						switch (d.Result) {
						case "added","modified":
							ddl, err := lib.GetColumnsDDL(servers, Config, d)
							if err != nil {
								log.Errorf("Failed to Get DDL %s.%s from %s",d.Table,d.Column,d.Db)
							}
							diffData[i].Ddl = ddl
						case "dropped":
							diffData[i].Ddl = fmt.Sprintf("DROP Columns %s.%s",d.Table,d.Column)
						}
					}
				}
				// Notification DDL Diff
				log.Infof("%s Check Definition Complete.",db)

				var colStr []string 
				var tableStr []string


				if len(diffData) != 0 {
					for _, v := range diffData {
						strFormat := "Status : *%s* Desc: `%s`"
						switch (v.Div) {
						case "table":
							tableStr = append(tableStr,fmt.Sprintf(strFormat,v.Result,v.Ddl))
							
						case "column":
							colStr = append(colStr,fmt.Sprintf(strFormat,v.Result,v.Ddl))
						}
					}
					if len(tableStr) != 0 {
						msg := fmt.Sprintf(slackFormat,servers.Alias,servers.Host,db,"Tables",strings.Join(tableStr,"\n"))
						slack.PostMessage(Config.Tracer.Webhook, msg)
					}

					if len(colStr) != 0 {
						
						msg := fmt.Sprintf(slackFormat,servers.Alias,servers.Host,db,"Columns",strings.Join(colStr,"\n"))
						slack.PostMessage(Config.Tracer.Webhook, msg)
					}

				}
			}
		}
		
	default:
		log.Errorf("Check", err)
	}
}