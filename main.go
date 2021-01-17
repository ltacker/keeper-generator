package main

import (
	"bytes"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const indexFieldNumber = 53535

// headerTemplate returns package and import statement
func headerTemplate(repoName string, projectName string, moduleName string) string {
	return fmt.Sprintf(`
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/%[1]v/%[2]v/x/%[3]v/types"
)
`, repoName, projectName, moduleName)
}

// setFunctionTemplate returns the template of a set function
func setFunctionTemplate(typeName string, indexName string) string {
	return fmt.Sprintf(`
// Set%[1]v set a %[1]v in the store
func (k Keeper) Set%[1]v(ctx sdk.Context, new%[1]v types.%[1]v) {
	store := ctx.KVStore(k.storeKey)
	bz := Marshal%[1]v(k.cdc, new%[1]v)
	store.Set(Get%[1]vKey(new%[1]v.%[2]v), bz)
}
`, typeName, indexName)
}

// getFunctionTemplate returns the template of a set function
func getFunctionTemplate(typeName string) string {
	return fmt.Sprintf(`
// Get%[1]v retrieve a %[1]v from the store
func (k Keeper) Get%[1]v(ctx sdk.Context, index string) (ret types.%[1]v, found bool) {
	store := ctx.KVStore(k.storeKey)

	value := store.Get(Get%[1]vKey(index))
	if value == nil {
		return ret, false
	}
	ret = Unmarshal%[1]v(k.cdc, value)

	return ret, true
}
`, typeName)
}

// marschalTemplate
func marschalTemplate(typeName string) string {
	return fmt.Sprintf(`
// Marshal%[1]v encodes %[1]vs for the store
func Marshal%[1]v(cdc codec.BinaryMarshaler, value types.%[1]v) []byte {
	return cdc.MustMarshalBinaryBare(&value)
}
`, typeName)
}

// unmarschalTemplate
func unmarschalTemplate(typeName string) string {
	return fmt.Sprintf(`
// Unmarshal%[1]v decodes %[1]vs from the store
func Unmarshal%[1]v(cdc codec.BinaryMarshaler, value []byte) types.%[1]v {
	var ret types.%[1]v
	cdc.MustUnmarshalBinaryBare(value, &ret)
	return ret
}
`, typeName)
}

// getKeyTemplate
func getKeyTemplate(typeName string) string {
	return fmt.Sprintf(`
// Get%[1]vKey returns the key for the %[1]v store
func Get%[1]vKey(index string) []byte {
	return append([]byte("IndexedTypes-%[1]v-"), []byte(index)...)
}
`, typeName)
}

// parseIndex search for the index option in a options string
func parseIndex(optionsStr string) (string, bool, error) {
	// Separate options
	options := strings.Split(optionsStr, " ")

	for _, option := range options {
		// Get field number and value
		fieldAndValue := strings.Split(option, ":")

		if len(fieldAndValue) != 2 {
			return "", false, fmt.Errorf("incorrect option: %v", option)
		}

		// Parse the field
		fieldNumber, err := strconv.Atoi(fieldAndValue[0])
		if err != nil {
			return "", false, err
		}
		if fieldNumber == indexFieldNumber {
			// Parse the value
			indexValue := fieldAndValue[1]
			if len(indexValue) < 3 {
				return "", false, fmt.Errorf("incorrect index: %v", indexValue)
			}

			// Remove quote
			index := indexValue[1 : len(indexValue)-1]
			return index, true, nil
		}
	}

	return "", false, nil
}

// generateCode generates codes from protobuf definition
func generateCode(req *pluginpb.CodeGeneratorRequest, res *pluginpb.CodeGeneratorResponse) error {
	var keeperFile pluginpb.CodeGeneratorResponse_File
	filename := "keeper.pb.go"
	keeperFile.Name = &filename
	var buf bytes.Buffer

	// Initialize the file header
	if _, err := buf.WriteString(headerTemplate(
		"titi",
		"toto",
		"tata",
	)); err != nil {
		return nil
	}

	// Iterate files
	for _, protofile := range req.GetProtoFile() {
		desc := protofile.GetSourceCodeInfo()
		locations := desc.GetLocation()

		// Iterate locations
		for _, location := range locations {

			// Parse message
			if len(location.GetPath()) == 2 && location.GetPath()[0] == int32(4) {
				message := protofile.GetMessageType()[location.GetPath()[1]]
				typeName := message.GetName()

				// Check the message has options
				if options := message.GetOptions(); options != nil {

					// Search for the index option
					indexName, found, err := parseIndex(options.String())
					if err != nil {
						return err
					}
					if found {
						// os.Stderr.WriteString(fmt.Sprintf("%v\n", indexOption))

						// Generate methods
						if _, err := buf.WriteString(getKeyTemplate(
							typeName,
						)); err != nil {
							return nil
						}
						if _, err := buf.WriteString(unmarschalTemplate(
							typeName,
						)); err != nil {
							return nil
						}
						if _, err := buf.WriteString(marschalTemplate(
							typeName,
						)); err != nil {
							return nil
						}
						if _, err := buf.WriteString(getFunctionTemplate(
							typeName,
						)); err != nil {
							return nil
						}
						if _, err := buf.WriteString(setFunctionTemplate(
							typeName,
							indexName,
						)); err != nil {
							return nil
						}
					}
				}
			}

		}
	}

	// Save generated file content
	content := buf.String()
	keeperFile.Content = &content
	res.File = append(res.File, &keeperFile)

	return nil
}

func main() {
	req := &pluginpb.CodeGeneratorRequest{}
	res := &pluginpb.CodeGeneratorResponse{}

	// Read input from protobuf
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	// Unmarschal request
	if err := proto.Unmarshal(data, req); err != nil {
		panic(err)
	}

	// Generate code
	res.File = make([]*pluginpb.CodeGeneratorResponse_File, 0)
	if err := generateCode(req, res); err != nil {
		panic(err)
	}

	// Write out proto marshalled response
	marshalled, err := proto.Marshal(res)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(marshalled)
}
