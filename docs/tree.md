## Project File Tree


```.
├── Dockerfile     
├── LICENSE        
├── README.md               // Readme English version  
├── README_ZH.md            // Readme Chinese version     
├── build.sh                // build script
├── cmd
│   ├── main.go    
│   ├── run.py              // Observer service startup file
│   ├── data_sources.go     // create data source connection
│   ├── injection.go        // inject data source to internal service / handler
│   ├── observer.go         // Observer service
├── configs        
│   ├── http.toml           // http server port config
│   ├── mysql.toml          // mysql config file    
│   ├── redis.toml          // redis config file
│   └── source              // custom types json 
│       └── crab.json
├── docker-compose.db.yml   // mysql.redis docker-compose file
├── docker-compose.yml      // subscan services docker-compose file
├── docs                    // docs dir
│   └── index.md
├── internal                
│   ├── repository          // data access service, used for db, redis resource access
│   ├── middleware          // http middleware
│   ├── script              // some script
│   ├── server                  
│   │   └── http            // init http server router 
│   └── service             // used for business logic processing
├── log                     // logs file dir
├── model                   // db table model
├── ui                      // Front-end code
└── util                    // some tools function
```