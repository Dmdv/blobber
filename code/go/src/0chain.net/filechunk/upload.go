package filechunk

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/klauspost/reedsolomon"
)

func uploadFile(filename string, reader io.Reader, wg *sync.WaitGroup, meta string) error {
	defer wg.Done()
	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)
	go func() {
		// fmt.Println("body buffer", bodyWriter)

		// this step is very important

		fileWriter, err := multiWriter.CreateFormFile("uploadFile", filename)
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}

		//iocopy
		_, err = io.Copy(fileWriter, reader)
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}

		// Create a form field writer for field label
		metaWriter, err := multiWriter.CreateFormField("custom_meta")
		if err != nil {
			bodyWriter.CloseWithError(err)
			return
		}
		metaWriter.Write([]byte(meta))

		bodyWriter.CloseWithError(multiWriter.Close())
	}()
	contentType := multiWriter.FormDataContentType()
	targetUrl := "http://localhost:5050/v1/file/upload/36f028580bb02cc8272a9a020f4200e346e276ae664e45ee80745574e2f5ab80"
	resp, err := http.Post(targetUrl, contentType, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err

	}
	fmt.Println("resp", string(resp_body))
	return nil
}

func storeInFile(in io.Reader, i int, wg *sync.WaitGroup) {
	defer wg.Done()
	destfilename := fmt.Sprintf("%s.%d", "big.txt", i)
	fmt.Println("file to be created", destfilename)
	f, err := os.Create(destfilename)
	defer f.Close()
	checkErr(err)
	// copy from reader data into writer file
	_, err = io.Copy(f, in)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("file created", destfilename)
}

//ChunkingFilebyShards is used to divide the file in chunks using erasure coding
func (fi *FileInfo) ChunkingFilebyShards() {
	runtime.GOMAXPROCS(30)
	if fi.DataShards > 257 {
		fmt.Fprintf(os.Stderr, "Error: Too many data shards\n")
		os.Exit(1)
	}
	fname := fi.File

	// Create encoding matrix.
	enc, err := reedsolomon.NewStreamC(fi.DataShards, fi.ParShards, true, true)
	checkErr(err)

	fmt.Println("Opening", fname)
	f, err := os.Open(fname)
	checkErr(err)

	instat, err := f.Stat()
	checkErr(err)

	shards := fi.DataShards
	var wg sync.WaitGroup
	wg.Add(18)

	out1 := make([]io.Writer, shards)
	out2 := make([]io.Writer, shards)
	out := make([]io.Writer, shards)

	in := make([]io.Reader, shards)
	inr := make([]io.Reader, shards)
	// Create the resulting files.

	for i := range out {
		outfn := fmt.Sprintf("Part : %d", i)
		meta := fmt.Sprintf("{\"part_num\" : %d}", i)
		fmt.Println("Creating", outfn)
		pr, pw := io.Pipe()
		npr, npw := io.Pipe()
		out1[i] = pw
		out2[i] = npw
		out[i] = io.MultiWriter(pw, npw)
		//out[i] = pw
		checkErr(err)
		//tr := io.TeeReader(pr, f)
		in[i] = pr
		inr[i] = npr
		//destfilename := fmt.Sprintf("%s.%d", "big.txt", i)
		go uploadFile(fname, npr, &wg, meta)
		//go storeInFile(npr, i, &wg);
	}

	// Create parity output writers
	parity := make([]io.Writer, 6)
	for i := range parity {
		// destfilename := fmt.Sprintf("%s.%d", "big.txt", 10+i)
		// fmt.Println("file to be created" , destfilename)
		// f, err := os.Create(destfilename)
		// defer f.Close()
		// checkErr(err)
		// parity[i] = f
		// //parity[i] = out[10+i]
		// //defer out[10+i].(*io.PipeWriter).Close()
		// fmt.Println("file created" , destfilename)
		pr, pw := io.Pipe()
		parity[i] = pw
		//destfilename := fmt.Sprintf("%s.%d", "big.txt", 10+i)
		meta := fmt.Sprintf("{\"part_num\" : %d}", 10+i)
		go uploadFile(fname, pr, &wg, meta)

	}

	go func() {
		defer wg.Done()
		// Encode parity
		err = enc.Encode(in, parity)
		checkErr(err)
		for i := range parity {
			parity[i].(*io.PipeWriter).Close()
		}
	}()

	go func() {
		defer wg.Done()
		// Do the split
		err = enc.Split(f, out, instat.Size())
		checkErr(err)
		fmt.Println("Done with split")
		for i := range out {
			out2[i].(*io.PipeWriter).Close()
			out1[i].(*io.PipeWriter).Close()
			//out2[i].(*io.PipeWriter).Close()
			//out[i].(*io.PipeWriter).Close()

		}
	}()

	wg.Wait()

	fmt.Printf("File split into %d data + %d parity shards.\n", 10, 6)
}

func Upload() {
	var fileinfo = &FileInfo{DataShards: 10,
		ParShards: 6,
		OutDir:    "",
		File:      "big.txt"}
	fileinfo.ChunkingFilebyShards()
}