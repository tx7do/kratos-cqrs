package v1

// 生成 proto
//go:generate protoc -I=. -I=../../../../../third_party --go_out=paths=source_relative:../ ./*.proto

// 生成 proto grpc
//go:generate protoc -I=. -I=../../../../../third_party --go-grpc_out=paths=source_relative:../ ./*.proto

// 生成 proto errors
//go:generate protoc -I=. -I=../../../../../third_party --go-errors_out=paths=source_relative:../ ./*.proto

// 生成 swagger
//go:generate protoc -I=. -I=../../../../../third_party --openapiv2_out=../ ./*.proto --openapiv2_opt logtostderr=true --openapiv2_opt json_names_for_fields=false
