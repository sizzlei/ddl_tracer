package lib

type Conf struct {
	Tracer		Tracer_conf	`yaml:"tracker"`
	Servers 	[]Server_conf	`yaml:"servers"`
}

type Tracer_conf struct {
	User		string  `yaml:"user"`
	Pass 		string	`yaml:"password"`
	Datapath	string 	`yaml:"datapath"`
	Webhook		string	`yaml:"webhook_url"`
	Tableddl	bool	`yaml:"tableddl"`
}


type Server_conf struct {
	Alias 		string 	`yaml:"alias"`
	Host 		string 	`yaml:"host"`
	Port 		int 	`yaml:"port"`
	Db 			[]string `yaml:"db"`
	HostPath 	string
}

type Diff_conf struct {
	Db 		string 
	Div 	string  // Div : table / Columns
	Table 	string
	Column	string
	Result	string
	Ddl 	string
}