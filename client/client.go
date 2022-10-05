package main

import (
	"EkSukkel/moeggesukkel"
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"google.golang.org/grpc"

	// "github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var log = flogging.MustGetLogger("MAIN")
var defaultBufSize int = 4096

func upload(address, path string, ttl int64) string {

	// Create client and
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Panic("failed to create gRPC connection")
	}

	mgc := moeggesukkel.NewMoegGeSukkelClient(conn)

	// client(address)

	// Stream and error
	ctx := context.Background()
	stream, err := mgc.Upload(ctx)

	if err != nil {
		log.Panic("Error: ", err)
	}

	// File
	// tar + gzip
	// var buf bytes.Buffer
	compressFileName := compress(path)
	// err = compress(path, &buf)

	if err != nil {
		log.Panic("Error: ", err)
	}

	fileToWrite, err := os.Open(compressFileName)

	if err != nil {
		log.Panic("Error: ", err)
	}

	// buffered reader
	dataArr := make([]byte, defaultBufSize)
	buffReader := bufio.NewReader(fileToWrite)
	fileInfo, err := fileToWrite.Stat()

	if err != nil {
		log.Panic("Error: ", err)
	}

	fileName := fileInfo.Name()

	for {
		_, readFinsihed := buffReader.Read(dataArr)

		request := moeggesukkel.UploadRequest{
			Filename: fileName,
			Time:     ttl,
			Data:     dataArr,
		}

		err = stream.Send(&request)

		if err != nil {
			log.Panic("Error: ", err)
		}

		if readFinsihed == io.EOF {
			// there is no more data to read
			break
			// return err
		}
	}

	// close file
	err = fileToWrite.Close()
	if err != nil {
		log.Panic("Error: ", err)
	}

	response, err := stream.CloseAndRecv()

	if err != nil {
		log.Panic("Error: ", err)
	}

	// log.Info("Deleting zip file : ", compressFileName)
	err = os.Remove(compressFileName)

	if err != nil {
		log.Panic("Error: ", err)
	}

	return response.GetToken()
}

func download(address, token string) {
	// Create client and
	// client(address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Panic("failed to create gRPC connection")
	}

	mgc := moeggesukkel.NewMoegGeSukkelClient(conn)

	// Stream and error
	ctx := context.Background()

	// request
	request := moeggesukkel.DownloadRequest{
		Token: token,
	}

	stream, err := mgc.Download(ctx, &request)

	if err != nil {
		log.Panic("Error: ", err)
	}

	firstResponse, err := stream.Recv()

	if err != nil {
		log.Panic("Error: ", err)
	}

	homeDir, _ := os.UserHomeDir()
	downloadFileName := filepath.Join(homeDir, "Downloads", firstResponse.GetFilename())
	fileToDownload, err := os.Create(downloadFileName)

	if err != nil {
		log.Panic("Error: ", err)
	}

	buffWriter := bufio.NewWriter(fileToDownload)
	_, err = buffWriter.Write(firstResponse.GetData())

	if err != nil {
		log.Panic("Error: ", err)
	}

	for {

		res, err := stream.Recv()

		if err != nil {
			break
		}

		_, err = buffWriter.Write(res.GetData())

		if err != nil {
			log.Panic("Error: ", err)
		}
	}

	err = buffWriter.Flush()
	if err != nil {
		log.Panic("Error: ", err)
	}

	err = fileToDownload.Close()
	if err != nil {
		log.Panic("Error: ", err)
	}
}

func compress(path string) string {

	fileBase := filepath.Base(path) + ".zip"

	// log.Info("Creating zip file : ", fileBase)

	err := ZipSource(path, fileBase)
	if err != nil {
		log.Panic("Error: ", err)
	}
	//  else {
	// 	log.Info("Zip file created at : ", fileBase)
	// }

	return fileBase

}

// write the .tar.gzip
// compressFileName := path + ".tar.gzip"
// fileToWrite, err := os.Create(compressFileName)

// if err != nil {
// 	log.Panic("Error: ", err)
// }

// if _, err := io.Copy(fileToWrite, &buf); err != nil {
// 	log.Panic("Error: ", err)
// }

// err = fileToWrite.Close()
// if err != nil {
// 	log.Panic("Error: ", err)
// }

// targetTar, err := Tar(path, "./")

// if err != nil {
// 	log.Panic("Error: ", err)
// } else {
// 	log.Info("Tar filed created at : ", targetTar)
// }

// compressTar, err := Gzip(targetTar, "./")

// if err != nil {
// 	log.Panic("Error: ", err)
// } else {
// 	log.Info("Gzip filed created at : ", compressTar)
// }

// return compressTar
// }

func Tar(source, target string) (string, error) {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar", filename))
	tarfile, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return "", nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	err = filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})

	if err != nil {
		return "", nil
	} else {
		return target, nil
	}
}

func Untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func Gzip(source, target string) (string, error) {
	reader, err := os.Open(source)
	if err != nil {
		return "", err
	}

	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.gz", filename))
	writer, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	archiver := gzip.NewWriter(writer)
	archiver.Name = filename
	defer archiver.Close()

	_, err = io.Copy(archiver, reader)

	if err != nil {
		return "", err
	} else {
		return target, nil
	}
}

func UnGzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}

func ZipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// func compress(src string, buf io.Writer) error {
// 	// tar > gzip > buf
// 	zr := gzip.NewWriter(buf)
// 	tw := tar.NewWriter(zr)

// 	// is file a folder?
// 	fi, err := os.Stat(src)
// 	if err != nil {
// 		return err
// 	}
// 	mode := fi.Mode()
// 	if mode.IsRegular() {
// 		// get header
// 		header, err := tar.FileInfoHeader(fi, src)
// 		if err != nil {
// 			return err
// 		}
// 		// write header
// 		if err := tw.WriteHeader(header); err != nil {
// 			return err
// 		}
// 		// get content
// 		data, err := os.Open(src)
// 		if err != nil {
// 			return err
// 		}
// 		if _, err := io.Copy(tw, data); err != nil {
// 			return err
// 		}
// 	} else if mode.IsDir() { // folder

// 		// walk through every file in the folder
// 		filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
// 			// generate tar header
// 			header, err := tar.FileInfoHeader(fi, file)
// 			if err != nil {
// 				return err
// 			}

// 			// must provide real name
// 			// (see https://golang.org/src/archive/tar/common.go?#L626)
// 			header.Name = filepath.ToSlash(file)

// 			// write header
// 			if err := tw.WriteHeader(header); err != nil {
// 				return err
// 			}
// 			// if not a dir, write file content
// 			if !fi.IsDir() {
// 				data, err := os.Open(file)
// 				if err != nil {
// 					return err
// 				}
// 				if _, err := io.Copy(tw, data); err != nil {
// 					return err
// 				}
// 			}
// 			return nil
// 		})
// 	} else {
// 		return fmt.Errorf("error: file type not supported")
// 	}

// 	// produce tar
// 	if err := tw.Close(); err != nil {
// 		return err
// 	}
// 	// produce gzip
// 	if err := zr.Close(); err != nil {
// 		return err
// 	}
// 	//
// 	return nil
// }

// // check for path traversal and correct forward slashes
// func validRelPath(p string) bool {
// 	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
// 		return false
// 	}
// 	return true
// }

// func decompress(src io.Reader, dst string) error {
// 	// ungzip
// 	zr, err := gzip.NewReader(src)
// 	if err != nil {
// 		return err
// 	}
// 	// untar
// 	tr := tar.NewReader(zr)

// 	// uncompress each element
// 	for {
// 		header, err := tr.Next()
// 		if err == io.EOF {
// 			break // End of archive
// 		}
// 		if err != nil {
// 			return err
// 		}
// 		target := header.Name

// 		// validate name against path traversal
// 		if !validRelPath(header.Name) {
// 			return fmt.Errorf("tar contained invalid name error %q", target)
// 		}

// 		// add dst + re-format slashes according to system
// 		target = filepath.Join(dst, header.Name)
// 		// if no join is needed, replace with ToSlash:
// 		// target = filepath.ToSlash(header.Name)

// 		// check the type
// 		switch header.Typeflag {

// 		// if its a dir and it doesn't exist create it (with 0755 permission)
// 		case tar.TypeDir:
// 			if _, err := os.Stat(target); err != nil {
// 				if err := os.MkdirAll(target, 0755); err != nil {
// 					return err
// 				}
// 			}
// 		// if it's a file create it (with same permission)
// 		case tar.TypeReg:
// 			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
// 			if err != nil {
// 				return err
// 			}
// 			// copy over contents
// 			if _, err := io.Copy(fileToWrite, tr); err != nil {
// 				return err
// 			}
// 			// manually close here after each file operation; defering would cause each file close
// 			// to wait until all operations have completed.
// 			fileToWrite.Close()
// 		}
// 	}

// 	//
// 	return nil
// }

func main() {
	// Silent
	var silent bool

	// TTL
	var ttl int64

	// Address
	var address string

	// Other
	var other string

	// Commands
	rootCmd := &cobra.Command{
		Use:   "moeggesukkel",
		Short: "gRPC client for a gRPC server",
		Long:  "Moeggesukkel is a gRPC client that connects to a gRPC server",
	}

	// Download
	downloadCmd := &cobra.Command{
		Use:   "download [address] [token]",
		Short: "downloads the given token from the server",
		Long:  "Passes the token to the Moeggesukkel server to download",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			if len(args) != 2 {
				log.Error("Incorrect arguments: provide only [address] & [token]")
			} else {
				address = args[0]
				other = args[1]
				log.Info("[address]: ", address, "  [token]: ", other)
				download(address, other)
			}

			// Download function
		},
	}

	downloadCmd.Flags().BoolVarP(&silent, "silent", "s", false, "show progress bars")

	// Upload
	uploadCmd := &cobra.Command{
		Use:   "upload [address] [path/to/file]",
		Short: "uploads the given file or directory to the server",
		Long:  "Streams the file or directory to the Moeggesukkel server and prints the tokens",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			if len(args) != 2 {
				log.Error("Incorrect arguments: provide only [address] & [path/to/file]")
			} else {
				address = args[0]
				other = args[1]
				log.Info("[address]: ", address, "  [path/to/file]: ", other)
				token := upload(address, other, ttl)
				log.Info("[token] : ", token)
			}

		},
	}

	uploadCmd.Flags().Int64VarP(&ttl, "ttl", "t", 3600, "amount of seconds that the should token is valid")

	rootCmd.AddCommand(downloadCmd, uploadCmd)
	rootCmd.Execute()

}

// func upload2(address, path string) error {

// 	// tar + gzip
// 	var buf bytes.Buffer

// 	err := compress(path, &buf)
// 	if err != nil {
// 		return err
// 	}
// 	// write the .tar.gzip
// 	fileToWrite, err := os.OpenFile(path+".tar.gzip", os.O_CREATE|os.O_RDWR, os.FileMode(600))

// 	if err != nil {
// 		return err
// 	}

// 	if _, err := io.Copy(fileToWrite, &buf); err != nil {
// 		return err
// 	}

// 	// untar write
// 	if err := untar(&buf, "./uncompressHere/"); err != nil {
// 		// probably delete uncompressHere?
// 	}

// }

// bar := progressbar.Default(100)
// for i := 0; i < 100; i++ {
// 	bar.Add(1)
// 	time.Sleep(40 * time.Millisecond)
// }
// writer := ansi.NewAnsiStdout()

// log.Info("Wrting to the thigny, starting now")

// bar1 := progressbar.NewOptions(1000,
// 	progressbar.OptionFullWidth(),
// 	progressbar.OptionSetWriter(writer),
// 	progressbar.OptionEnableColorCodes(true),
// 	progressbar.OptionShowBytes(true),
// 	progressbar.OptionSetWidth(15),
// 	progressbar.OptionSetDescription("[cyan][1/3][reset] Writing moshable file..."),
// 	progressbar.OptionSetTheme(progressbar.Theme{
// 		Saucer:        "[green]=[reset]",
// 		SaucerHead:    "[green]>[reset]",
// 		SaucerPadding: " ",
// 		BarStart:      "[",
// 		BarEnd:        "]",
// 	}))
// for i := 0; i < 1000; i++ {
// 	bar1.Add(1)
// 	time.Sleep(5 * time.Millisecond)
// }

// fmt.Println()
// log.Info("\nWrting to the thigny, starting now")

// bar2 := progressbar.NewOptions(1000,
// 	progressbar.OptionFullWidth(),
// 	progressbar.OptionSetWriter(writer),
// 	progressbar.OptionEnableColorCodes(true),
// 	progressbar.OptionShowBytes(true),
// 	progressbar.OptionSetWidth(15),
// 	progressbar.OptionSetDescription("[yellow][2/3] Second detached stage..."),
// 	progressbar.OptionSetTheme(progressbar.Theme{
// 		Saucer:        "[green]=[reset]",
// 		SaucerHead:    "[green]>[reset]",
// 		SaucerPadding: " ",
// 		BarStart:      "[",
// 		BarEnd:        "]",
// 	}))
// for i := 0; i < 1000; i++ {
// 	bar2.Add(2)
// 	time.Sleep(5 * time.Millisecond)
// }
// fmt.Println()
// log.Info("\nWrting to the thigny, starting now")

// bar := progressbar.NewOptions(1000,
// 	progressbar.OptionFullWidth(),
// 	progressbar.OptionSetWriter(writer),
// 	progressbar.OptionEnableColorCodes(true),
// 	progressbar.OptionShowBytes(true),
// 	progressbar.OptionSetWidth(15),
// 	progressbar.OptionSetDescription("[red][3/3] Deploying files..."),
// 	progressbar.OptionSetTheme(progressbar.Theme{
// 		Saucer:        "[green]=[reset]",
// 		SaucerHead:    "[green]>[reset]",
// 		SaucerPadding: " ",
// 		BarStart:      "[",
// 		BarEnd:        "]",
// 	}))
// for i := 0; i < 1000; i++ {
// 	bar.Add(10)
// 	time.Sleep(5 * time.Millisecond)
// }

// io.MultiWriter()
