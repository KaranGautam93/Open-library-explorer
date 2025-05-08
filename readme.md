Open library explorer

pre-requisites
golang >= 1.18
mongo >= 6.X

to setup
create db "library" in mongo and add 
.env file in root of project

execute following commands once connected to mongodb

****
- use library;
- db.books.createIndex({ isbn: 1 }, { unique: true });
- db.copies.createIndex({ barcode: 1 }, { unique: true });
- db.books.createIndex(
{ title: "text", author: "text", subject: "text" },
{ name: "TextIndex" }
)

to start server run following command from root of project
- go run cmd/main.go

to execute test case run following command from root of project
- go test -v ./...