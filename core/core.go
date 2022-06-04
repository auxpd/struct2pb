package core

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"time"
)

const (
	// indent represents the indentation amount for fields. the style guide suggests
	// two spaces
	indent = "  "

	structStart = "type"
	structEnd   = "}"
	fieldSep    = " "
	commentSep  = "//"
)

// MessageField represents the field of a message.
type MessageField struct {
	Typ     string
	Name    string
	tag     int
	Comment string
}

// NewMessageField creates a new message field.
func NewMessageField(typ, name string, tag int, comment string) MessageField {
	return MessageField{typ, name, tag, comment}
}

// Tag returns the unique numbered tag of the message field.
func (f MessageField) Tag() int {
	return f.tag
}

// String returns a string representation of a message field.
func (f MessageField) String() string {
	return fmt.Sprintf("%s %s = %d", f.Typ, f.Name, f.tag)
}

// Message represents a protocol buffer message.
type Message struct {
	Name    string
	Comment string
	Fields  []MessageField
}

// String returns a string representation of a Message.
func (m Message) String() string {
	var buf bytes.Buffer

	if len(m.Comment) > 0 {
		buf.WriteString(fmt.Sprintf("// %s\n", m.Comment))
	}
	buf.WriteString(fmt.Sprintf("message %s {\n", m.Name))
	for _, f := range m.Fields {
		if len(f.Comment) > 0 {
			buf.WriteString(fmt.Sprintf("%s%s; // %s\n", indent, f, f.Comment))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s;\n", indent, f))
		}

	}
	buf.WriteString("}\n")

	return buf.String()
}

var (
	pbFloat64 = "double"
	pbFloat32 = "float"
	pbInt64   = "int64"
	pbInt32   = "int32"
	pbUint64  = "uint64"
	pbUint32  = "uint32"
	pbBool    = "bool"
	pbString  = "string"
	pbArray   = "repeated"
	pbMap     = "map"
	pbAny     = "Any"
)

func Structs2Pb(strictMode bool, beans ...interface{}) string {
	var result string
	for i := range beans {
		bean := beans[i]
		// 获取结构体的反射类型对象
		v := reflect.Indirect(reflect.ValueOf(bean))
		vT := v.Type()

		comment, fields := struct2PbField(vT, 1, strictMode)
		message := Message{
			Name:    vT.Name(),
			Comment: comment,
			Fields:  fields,
		}
		result += message.String() + string('\n')
	}
	return result
}

func struct2PbField(t reflect.Type, index int, strictMode bool) (comment string, fields []MessageField) {
	c, fieldMap, err := getStructComment(t)
	if err != nil {
		panic(err)
	}
	comment = c

	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		// 忽略未导出字段
		if importable := len(fieldType.PkgPath) == 0; !importable {
			continue
		}
		// 匿名字段
		if fieldType.Anonymous {
			_, newFields := struct2PbField(fieldType.Type.Elem(), index, strictMode)
			index += len(newFields)
			fields = append(fields, newFields...)
			continue
		}
		pbType := goType2PbType(fieldType.Type, strictMode)
		fieldName := Camel2CamelLower(fieldType.Name)
		fieldComment := fieldMap[fieldType.Name]
		fields = append(fields, NewMessageField(pbType, fieldName, index, fieldComment))

		index++
	}
	return
}

// goType2PbType go type to pb type
func goType2PbType(t reflect.Type, strictMode bool) string {
	// var cByteDefault byte
	timeType := reflect.TypeOf(time.Time{})
	// byteType := reflect.TypeOf(cByteDefault)
	// bytesType := reflect.SliceOf(byteType)
	switch k := t.Kind(); k {
	case reflect.Float64:
		return pbFloat64
	case reflect.Float32:
		return pbFloat32

	case reflect.Int:
		fallthrough
	case reflect.Int64:
		return pbInt64
	case reflect.Int32:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int8:
		return pbInt32

	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		return pbUint64
	case reflect.Uint32:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint8:
		return pbUint32

	case reflect.Bool:
		return pbBool

	case reflect.String:
		return pbString

	case reflect.Slice:
		fallthrough
	case reflect.Array:
		value := goType2PbType(t.Elem(), strictMode)
		return pbArray + fieldSep + value

	case reflect.Map:
		var value string
		if !allowedMapKey(t.Key()) || !allowedMapValue(t.Elem()) {
			// TODO: 支持复杂类型
			if strictMode {
				panic(fmt.Sprintf("unsupported map type: key:%s  value:%s\n", t.Key().String(), t.Elem().String()))
			} else {
				value = pbAny
			}
		} else {
			value = goType2PbType(t.Elem(), strictMode)
		}
		return pbMap + "<" + t.Key().String() + ", " + value + ">"

	// case bytesType.Kind():
	// 	return "bytes"

	case reflect.Struct:
		// 时间类型
		if t.ConvertibleTo(timeType) {
			return pbInt64
		} else {
			// 其他struct
			return t.Name()
		}
	case reflect.Ptr:
		return goType2PbType(t.Elem(), strictMode)
	default:
		panic(fmt.Sprintf("unsupported type: %s\n", k.String()))
	}
}

func allowedMapValue(t reflect.Type) bool {
	// map字段不能使用repeated关键字修饰
	switch t.Kind() {
	case reflect.Map:
		return false
	case reflect.Array:
		return false
	case reflect.Slice:
		return false
	default:
		return true
	}
}

func allowedMapKey(t reflect.Type) bool {
	// 可以是任何证书或字符串类型（除浮点类型和字节之外的任何标量类型）、不能是枚举
	switch t.Kind() {
	case reflect.Map:
		fallthrough

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		fallthrough

	case reflect.Float64:
		fallthrough
	case reflect.Float32:
		return false
	default:
		return true
	}
}

// Camel2CamelLower big camel to small camel
func Camel2CamelLower(s string) string {
	a := strings.ToLower(string(s[0]))
	return a + s[1:]
}

// get comment for the structure
func getStructComment(vT reflect.Type) (string, map[string]string, error) {
	structName := vT.PkgPath() + "." + vT.Name()

	var fieldCommentMap = make(map[string]string)
	cmd := exec.Command("go", "doc", structName)
	output, err := cmd.Output()
	if err != nil {
		return "", nil, err
	}
	buf := bytes.NewBuffer(output)
	var (
		isEnd   bool
		comment string
	)
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}

		if strings.TrimSpace(line) == structEnd {
			isEnd = true
			continue
		}

		infoList := strings.Split(line, commentSep) // 拆分出注释行
		if len(infoList) == 0 {
			continue
		}
		keyList := strings.Split(strings.TrimSpace(infoList[0]), fieldSep)
		if !isEnd {
			if len(keyList) == 1 { // 匿名结构体
				continue
			}
			// 结构体定义头
			var fieldName = keyList[0]
			if fieldName == structStart {
				continue
			}
			// 字段定义有注释
			if len(keyList) >= 2 && len(infoList) >= 2 {
				var commentList []string
				for _, comment := range infoList[1:] {
					commentList = append(commentList, strings.TrimSpace(comment))
				}
				fieldCommentMap[fieldName] = strings.Join(commentList, " ")
			}
		} else {
			comment = strings.TrimSpace(line)
			break
		}
	}
	return comment, fieldCommentMap, nil
}
