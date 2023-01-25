package createworker

import (
	"bytes"
	//"crypto/des"
	gob "encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	b64 "encoding/base64"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"

	vault "github.com/sosedoff/ansible-vault-go"
)

func createDirectory(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.Mkdir(directory, 0777)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

func currentdir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return "", nil
	}

	return cwd, nil
}

func writeFile(filePath string, contentLines []string) error {
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return nil
	}
	for _, v := range contentLines {
		fmt.Fprintf(f, v)
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return nil
}

func encryptFile(filePath string, vaultPassword string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = vault.EncryptFile(filePath, string(data), vaultPassword)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return nil
}

func decryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	//fmt.Println("Decrypted: ")
	//fmt.Println(data)
	return data, nil
}

func generateB64Credentials(access_key string, secret_key string, region string) string {
	credentialsINIlines := "[default]\naws_access_key_id = " + access_key + "\naws_secret_access_key = " + secret_key + "\nregion = " + region + "\n\n"
	return b64.StdEncoding.EncodeToString([]byte(credentialsINIlines))
}

func getCredentials(descriptorFile cluster.DescriptorFile, vaultPassword string) (cluster.AWSCredentials, string, error) {
	aws := cluster.AWSCredentials{}

	_, err := os.Stat("./secrets.yaml")
	if err != nil {
		fmt.Println("descriptorFile.AWS: ", descriptorFile.AWSCredentials)
		if aws != descriptorFile.AWSCredentials {
			return descriptorFile.AWSCredentials, descriptorFile.GithubToken, nil
		}
		err := errors.New("Incorrect AWS credentials in Cluster.yaml")
		return aws, "", err

	} else {
		secretRaw, err := decryptFile("./secrets.yaml", vaultPassword)
		var secretFile SecretsFile
		if err != nil {
			err := errors.New("The vaultPassword is incorrect")
			return aws, "", err
		} else {
			fmt.Println("secretRAW: ")
			fmt.Println(secretRaw)
			err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
			if err != nil {
				fmt.Println(err)
				return aws, "", err
			}
			fmt.Println("secretFile: ", secretFile)
			fmt.Println("secretFile.Secret: ", secretFile.Secret)
			return secretFile.Secret.AWSCredentials, secretFile.Secret.GithubToken, nil
		}
	}

}

func stringToBytes(str string) []byte {
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(str)
	bytes := buf.Bytes()

	return bytes
}

func rewriteDescriptorFile() error {

	descriptorRAW, err := os.ReadFile("./cluster.yaml")
	if err != nil {
		return err
	}

	descriptorMap := map[string]interface{}{}
	viper.SetConfigName("cluster.yaml")
	currentDir, err := currentdir()
	if err != nil {
		fmt.Println(err)
		return err
	}
	viper.AddConfigPath(currentDir)

	//fmt.Println(descriptor)
	//descriptor = descriptorFile
	//descriptorFile.AWS = AWS{}

	err = yaml.Unmarshal(descriptorRAW, &descriptorMap)
	if err != nil {
		return err
	}

	fmt.Println("Before descriptorMap: ")
	fmt.Println(descriptorMap)

	// aws := descriptorMap["aws"]
	// if aws != nil {
	// 	delete(descriptorMap, "aws")
	// }
	deleteKey("aws", descriptorMap)
	deleteKey("github_token", descriptorMap)

	fmt.Println("After descriptorMap: ")
	fmt.Println(descriptorMap)

	d, err := yaml.Marshal(&descriptorMap)
	if err != nil {
		fmt.Println("error: %v", err)
		return err
	}

	//fmt.Println(string(d))

	// write to file
	f, err := os.Create(currentDir + "/cluster.yaml")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	err = ioutil.WriteFile("cluster.yaml", d, 0755)
	if err != nil {
		fmt.Println("error: %v", err)
		return err
	}

	f.Close()

	return nil

}

func deleteKey(key string, descriptorMap map[string]interface{}) {
	value := descriptorMap[key]
	if value != nil {
		delete(descriptorMap, key)
	}
}
