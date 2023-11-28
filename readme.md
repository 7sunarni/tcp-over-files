# TCP Over Files

Establish tcp connection through normal files.(not socket file)

## Why this
Let's say there are two computers, they shared a disk.

1. Computer1 open file1 READ_ONLY as input, open file2 READ_WRITE as output in sharing disk;
2. Computer2 open file2 READ_ONLY as input, open file1 READ_WRITE as output in sharing disk;
3. Computer1 dial tcp connection, and read file1 data then write to tcp connection, read tcp connection data then write to file2;
4. Computer2 listen local port, when there is connection, read connection data write to file1, also read file2 data write to tcp connection.

So, we just establish tcp connection through two common shared files.

## Local Example
1. terminal 1:
```shell
python3 -m http.server 12345 # prepare a example tcp server
```

2. terminal 2:
```shell
touch input; 
touch output;
go run main.go -type server -input input -output output -forward "12345"
```

3. terminal 3:
```shell
go run main.go -type client -input output -output input -listen "54321"
```

4. terminal 4:
```shell
curl http://localhost:54321 # you just view to step 1 python http server
```

