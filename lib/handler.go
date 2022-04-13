package lib


import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"os"
	"fmt"
	"io/ioutil"
	"strings"
)


const (
	dsnFormat = "%s:%s@tcp(%s:%d)/information_schema"
	readQuery = `
		select 
			table_name,
			CONCAT(column_name,":",
			MD5(concat_ws("|",
			is_nullable,
			IFNULL(column_default,"NULL"),
			IFNULL(character_set_name,"NULL"),
			IFNULL(collation_name,"NULL"),
			column_type,
			IF(extra="","NULL",extra),
			IF(column_comment="","NULL",column_comment)
			)))
		from information_schema.columns
		where table_schema="%s"
		order by table_name
	`
	readColQuery = `
		SELECT
		CONCAT(
			table_name,".",column_name," ",column_type," ",
			IF(character_set_name IS NULL,"",CONCAT("CHARACTER SET ",character_set_name," ")),
			IF(collation_name IS NULL,"",CONCAT("COLLATE ",collation_name," ")),
			IF(is_nullable="YES","NULL ", "NOT NULL "),
			IF(column_default IS NULL,"",if(column_default = "","DEFAULT ''",CONCAT("DEFAULT ",column_default," "))),
			IF(extra = "","",CONCAT(extra," ")),
			IF(column_comment = "", "", CONCAT("COMMENT '",column_comment,"'"))
			) 
		FROM information_schema.COLUMNS 
		WHERE table_schema = "%s"
			and table_name = "%s"
			and column_name = "%s";
	`
	readCreateQuery = "show create table %s.%s;"
	readTableQuery = "select table_name,concat('Table Name : ',table_name,' / Table Comment : ',table_comment) from information_schema.tables where table_schema='%s' and table_name='%s'"
)

func MakePath(Path string) error{
	err := os.Mkdir(Path,0750)
	if err != nil && !os.IsExist(err) {
		return err
	} 

	return err
}

func MakeFile(Path string,FileName string) error {
	err := ioutil.WriteFile(fmt.Sprintf("%s/%s",Path,FileName), []byte(""), 0750)
	if err != nil {
		return err
	}
	return nil
}

func AddContains(s []string,str string) (string,string){
	cName := strings.Split(str,":")
	for _, v := range s {
		fName := strings.Split(v,":")
		// Column Name Chcek
		if fName[0] == cName[0] {
			// Column definition Check
			if fName[1] != cName[1] {
				// Diff Definition
				return v,"m"
			}
			// Same Definition
			return str,"s"
		}
	}
	// Added Definition
	return str,"a"
}

func DropContains(s []string,str string) (string,string){
	cName := strings.Split(str,":")
	for _, v := range s {
		fName := strings.Split(v,":")
		// Column Name Chcek
		if fName[0] == cName[0] {
			return str,"s"
		}
	}
	// Added Definition
	return str,"d"
}

func SetInit(Server Server_conf, Config *Conf) error{
	// Make DSN
	DSN := fmt.Sprintf(
		dsnFormat,
		Config.Tracer.User,
		Crypto(0,Config.Tracer.Pass),
		Server.Host,
		Server.Port,
	)

	// Create DB Object
	dbObj, err := sql.Open("mysql",DSN)
	if err != nil {
		log.Errorf("Failed to create DB Object.")
		return err
	}
	defer dbObj.Close()

	for _, db := range Server.Db {
		// Setup DB Path
		dbPath := fmt.Sprintf("%s/%s",Server.HostPath, db)

		log.Infof("Setup DB Path : %s",dbPath)
		_ = os.RemoveAll(dbPath)
		err := MakePath(dbPath)
		if err != nil {
			log.Errorf("Failed to Setup Path.")
			return err
		}

		// Information_schema Read
		colData, err := dbObj.Query(fmt.Sprintf(readQuery,db))
		if err != nil {
			log.Errorf("Failed to DDL Read Query.")
			return err
		}

		for colData.Next() {
			var tbName, colName string 
			
			// Column Data Scan
			err := colData.Scan(&tbName,&colName)
			if err != nil {
				log.Errorf("Failed to Column data read.")
				return err
			}

			// Make Table Path
			tablePath := fmt.Sprintf("%s/%s",dbPath,tbName)
			_ = MakePath(tablePath)
			err = MakeFile(tablePath,colName)
			if err != nil {
				log.Errorf("Failed to Make Column Files.")
				return err
			}
		}
	}
	return nil
}

func GetColumns(Server Server_conf,db string) (map[string][]string,error){
	// Decalre Map
	var tbMap map[string][]string
	tbMap = make(map[string][]string)

	// Setup DB Path
	dbPath := fmt.Sprintf("%s/%s",Server.HostPath, db)

	// Get Table Directory
	tbDir, err := ioutil.ReadDir(dbPath)
	if err != nil {
		log.Errorf("Failed to get table directory")
		return tbMap,err
	}

	// Get Column File Info
	for _, tables := range tbDir {
		files, err := ioutil.ReadDir(fmt.Sprintf("%s/%s/",dbPath,tables.Name()))
		if err != nil {
			log.Errorf("Failed to get Column files")
			return tbMap,err
		}
		for _, file := range files{
			tbMap[tables.Name()] = append(tbMap[tables.Name()],file.Name())
		}
	}

	return tbMap, nil
}

func GetDefinition(Server Server_conf, db string, Config *Conf) (map[string][]string,error){
	// Decalre Map
	var dbMap map[string][]string
	dbMap = make(map[string][]string)

	// Make DSN
	DSN := fmt.Sprintf(
		dsnFormat,
		Config.Tracer.User,
		Crypto(0,Config.Tracer.Pass),
		Server.Host,
		Server.Port,
	)

	// Create DB Object
	dbObj, err := sql.Open("mysql",DSN)
	if err != nil {
		log.Errorf("Failed to create DB Object.")
		return dbMap,err
	}
	defer dbObj.Close()

	// Information_schema Read
	colData, err := dbObj.Query(fmt.Sprintf(readQuery,db))
	if err != nil {
		log.Errorf("Failed to DDL Read Query.")
		return dbMap,err
	}

	for colData.Next() {
		var tbName, colName string 
		// Column Data Scan
		err := colData.Scan(&tbName,&colName)
		if err != nil {
			log.Errorf("Failed to Column data read.")
			return dbMap,err
		}

		// Make DB Map
		dbMap[tbName] = append(dbMap[tbName],colName)

	}

	return dbMap,nil
}

func DiffMap(hostPath string, db string,fileMap map[string][]string,dbMap map[string][]string) []Diff_conf{
	var diffData []Diff_conf

	// DB Path
	dbPath := fmt.Sprintf("%s/%s",hostPath,db)
	
	// Diff ADD / MODIFY
	for k, v := range dbMap {
		// Table Path
		tbPath := fmt.Sprintf("%s/%s",dbPath,k)

		// CHeck Deifinitions
		if fileMap[k] == nil {
			// New Tables
			log.Errorf("%s Table has been added",k)

			// Create New Table Path
			_ = MakePath(tbPath)

			for _, c := range v {
				// Create New Col File
				_ = MakeFile(tbPath,c)
			}
			
			// Declare Diff Data
			var diffTables Diff_conf 
			diffTables.Db = db 
			diffTables.Div = "table"
			diffTables.Table = k
			diffTables.Result = "added"


			diffData = append(diffData,diffTables)


		} else {
			for _, c := range v {
				cols, r := AddContains(fileMap[k],c)
				switch (r) {
				case "s":
					// Same Columns
					continue
				case "m":
					// Different Columns
					log.Errorf("%s.%s has modified",k,cols)

					// Remove File Columns
					err := os.Remove(fmt.Sprintf("%s/%s",tbPath,cols))
					if err != nil {
						log.Errorf("Failed to File Remove %s.%s",tbPath,cols)
					}

					// Create New Columns 
					err = MakeFile(tbPath,c)
					if err != nil {
						log.Errorf("Failed to Create New file %s.%s",tbPath,cols)
					}

					// Declare Diff Data
					colsName := strings.Split(cols,":")

					var diffCols Diff_conf 
					diffCols.Db = db 
					diffCols.Div = "column"
					diffCols.Table = k
					diffCols.Column = colsName[0]
					diffCols.Result = "modified"


					diffData = append(diffData,diffCols)

				case "a":
					// Added Columns
					log.Errorf("%s.%s has been added",k,cols)
					// Create New Columns 
					err := MakeFile(tbPath,c)
					if err != nil {
						log.Errorf("Failed to Create New file %s.%s",tbPath,cols)
					}

					// Declare Diff Data
					colsName := strings.Split(cols,":")

					var addCols Diff_conf 
					addCols.Db = db 
					addCols.Div = "column"
					addCols.Table = k
					addCols.Column = colsName[0]
					addCols.Result = "added"

					diffData = append(diffData,addCols)
				}
			}
		}
	}

	// Diff DROP
	for k, v := range fileMap {
		// Table Path
		tbPath := fmt.Sprintf("%s/%s",dbPath,k)

		if dbMap[k] == nil {
			log.Errorf("%s Table has been dropped",k)

			// Drop Tables
			_ = os.RemoveAll(tbPath)

			// Declare Diff Data
			var dropTables Diff_conf 
			dropTables.Db = db 
			dropTables.Div = "table"
			dropTables.Table = k
			dropTables.Result = "dropped"

			diffData = append(diffData,dropTables)

		} else {
			for _, c := range v {
				cols, r := DropContains(dbMap[k],c)
				switch (r) {
				case "s":
					continue
				case "d":
					// Added Columns
					log.Errorf("%s.%s has been dropped",k,cols)
					// Create New Columns 
					err := os.Remove(fmt.Sprintf("%s/%s",tbPath,c))
					if err != nil {
						log.Errorf("Failed to Remove file %s.%s",tbPath,cols)
					}

					// Declare Diff Data
					colsName := strings.Split(cols,":")

					var dropCols Diff_conf 
					dropCols.Db = db 
					dropCols.Div = "column"
					dropCols.Table = k
					dropCols.Column = colsName[0]
					dropCols.Result = "dropped"

					diffData = append(diffData,dropCols)
				}
			}
		}
	}

	return diffData
}

func GetTablesDDL(Server Server_conf, Config *Conf,DiffTable Diff_conf) (string,error){
	var CreateTable string
	var CreateStr string
	// Make DSN
	DSN := fmt.Sprintf(
		dsnFormat,
		Config.Tracer.User,
		Crypto(0,Config.Tracer.Pass),
		Server.Host,
		Server.Port,
	)

	// Create DB Object
	dbObj, err := sql.Open("mysql",DSN)
	if err != nil {
		log.Errorf("Failed to create DB Object.")
		return CreateStr,err
	}
	defer dbObj.Close()

	if Config.Tracer.Tableddl == true {
		err := dbObj.QueryRow(fmt.Sprintf(readCreateQuery,DiffTable.Db,DiffTable.Table)).Scan(&CreateTable,&CreateStr)
		if err != nil {
			log.Errorf("Failed to DDL Read Query.")
			return CreateStr,err
		}
	} else {
		err := dbObj.QueryRow(fmt.Sprintf(readTableQuery,DiffTable.Db,DiffTable.Table)).Scan(&CreateTable,&CreateStr)
		if err != nil {
			log.Errorf("Failed to DDL Read Query.")
			return CreateStr,err
		}
	}
	return CreateStr,nil
}

func GetColumnsDDL(Server Server_conf, Config *Conf, DiffTable Diff_conf) (string,error){
	var ColumneStr string
	// Make DSN
	DSN := fmt.Sprintf(
		dsnFormat,
		Config.Tracer.User,
		Crypto(0,Config.Tracer.Pass),
		Server.Host,
		Server.Port,
	)

	// Create DB Object
	dbObj, err := sql.Open("mysql",DSN)
	if err != nil {
		log.Errorf("Failed to create DB Object.")
		return ColumneStr,err
	}
	defer dbObj.Close()

	err = dbObj.QueryRow(fmt.Sprintf(readColQuery,DiffTable.Db,DiffTable.Table,DiffTable.Column)).Scan(&ColumneStr)
	if err != nil {
		log.Errorf("Failed to Column DDL Read Query.")
		return ColumneStr,err
	}

	return ColumneStr,nil
}