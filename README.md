
## A tool to convert go structure to proto structure


### usage:
1. Pull project
2. Add go structure to **struct.go** in the **obj** directory and add it to the **List** object
3. Execute **struct2pb.go** and the conversion results will be printed to the **console**

### note:
- time.Time will be converted to int64 type
- In non-strict mode, unsupported types are converted to Any type

