package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SRUResponse struct {
	Records []Record `xml:"//record"`
}

type Record struct {
	Datafields []Datafield `xml:"//datafield"`
}

type Datafield struct {
	Tag       string     `xml:"tag,attr"`
	Subfields []Subfield `xml:"subfield"`
}

type Subfield struct {
	Code string `xml:"code,attr"`
	Text string `xml:",chardata"`
}

var counter = 0

func searchTGLSRUDeepParse(normNumber string) []string {
	var documentCollected []string = nil // make([]string, 1)
	//u, err := url.Parse("https://services.dnb.de/sru/dnb?maximumRecords=5&operation=searchRetrieve&query=tit%3D%22TGL+32565%22&recordSchema=MARC21-xml&version=1.1")
	u, err := url.Parse("https://services.dnb.de/sru/dnb")
	if err != nil {
		fmt.Printf("Error in base URL: %v\n", err)
		return nil
	}

	rawQuery := fmt.Sprintf(`tit="TGL %s"`, normNumber)
	q := u.Query()
	q.Set("version", "1.1")
	q.Set("operation", "searchRetrieve")
	q.Set("query", rawQuery)
	q.Set("recordSchema", "MARC21-xml")
	q.Set("maximumRecords", "5")
	u.RawQuery = q.Encode()

	fmt.Printf("Query API with URL:\n%s\n\n", u.String())

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Printf("❌ Error during querying: %v\n", err)
		return nil
	}

	//req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0")
	//req.Header.Set("Accept", "application/xml, text/xml, */*")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Network error: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	fmt.Printf("%v\n", resp.StatusCode)
	fmt.Printf("%v\n", resp.ContentLength)
	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("%v\n", contentType)

	if strings.Contains(contentType, "text/html") {
		fmt.Println("❌ unexpected text/html response (expected: text/xml)")
		return nil
	}

	// read in flat XML bytes
	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading in raw bytes of query result: %v\n", err)
		return nil
	}
	fmt.Printf("Raw bytes: %v\n", string(rawBytes))

	sruData := parseXMLNamespaceInsensitive(rawBytes)

	if len(sruData.Records) == 0 {
		fmt.Println("⚠️ no records found in XML response")
		return nil
	}

	fmt.Printf("🎉 %d records in XML response.\n\n", len(sruData.Records))

	for i, record := range sruData.Records {
		fmt.Printf("--- [data set %d] ---\n", i+1)
		title := "Not in default field (245)"
		var links []string
		var additionalInfo []string

		for _, df := range record.Datafields {
			subfieldMap := make(map[string]string)
			for _, sf := range df.Subfields {
				if sf.Text != "" {
					subfieldMap[sf.Code] = sf.Text
				}
			}

			switch df.Tag {
			case "245":
				var titleParts []string
				for _, code := range []string{"a", "b", "c"} {
					if val, ok := subfieldMap[code]; ok {
						titleParts = append(titleParts, val)
					}
				}
				if len(titleParts) > 0 {
					title = strings.Join(titleParts, " ")
				}
			case "856":
				if uLink, ok := subfieldMap["u"]; ok {
					links = append(links, uLink)
				}
			case "260", "264":
				if year, ok := subfieldMap["c"]; ok {
					year = strings.TrimRight(year, " .")
					if year != "" {
						additionalInfo = append(additionalInfo, "Year: "+year)
					}
				}
			case "500":
				if note, ok := subfieldMap["a"]; ok {
					if note != "" {
						additionalInfo = append(additionalInfo, "Remark: "+note)
					}
				}
			}
		}

		fmt.Printf("  Title: %s\n", title)
		if len(additionalInfo) > 0 {
			fmt.Printf("  Details: %s\n", strings.Join(additionalInfo, ", "))
		}
		if len(links) > 0 {
			fmt.Println("  🔗 Links found:")
			for _, link := range links {
				fmt.Printf("    -> %s\n", link)
				documentCollected = append(documentCollected, link)
			}
		} else {
			fmt.Println("  ⚠️ (No link inside this record)")
		}
		fmt.Println()
	}
	return documentCollected
}

func parseXMLNamespaceInsensitive(data []byte) SRUResponse {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var response SRUResponse
	var currentRecord *Record
	var currentField *Datafield

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "record":
				currentRecord = &Record{}
			case "datafield":
				if currentRecord != nil {
					var tag string
					for _, attr := range se.Attr {
						if attr.Name.Local == "tag" {
							tag = attr.Value
						}
					}
					currentField = &Datafield{Tag: tag}
				}
			case "subfield":
				if currentField != nil {
					var code string
					for _, attr := range se.Attr {
						if attr.Name.Local == "code" {
							code = attr.Value
						}
					}
					var text string
					if err := decoder.DecodeElement(&text, &se); err == nil {
						currentField.Subfields = append(currentField.Subfields, Subfield{Code: code, Text: text})
					}
				}
			}
		case xml.EndElement:
			switch se.Name.Local {
			case "datafield":
				if currentRecord != nil && currentField != nil {
					currentRecord.Datafields = append(currentRecord.Datafields, *currentField)
					currentField = nil
				}
			case "record":
				if currentRecord != nil {
					response.Records = append(response.Records, *currentRecord)
					currentRecord = nil
				}
			}
		}
	}
	return response
}

// ProgressReader implements io.Reader and count bytes read
// emits a star each 1 MByte data downloaded
type ProgressReader struct {
	Src       io.Reader
	BytesRead int64
	LastStar  int64
}

// Read calculates progress in read
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Src.Read(p)
	pr.BytesRead += int64(n)

	// 1 MByte = 1024 * 1024 Bytes
	const megaByte = 1024 * 1024

	// number of new stars
	currentMilestone := pr.BytesRead / megaByte
	lastMilestone := pr.LastStar / megaByte

	if currentMilestone > lastMilestone {
		starsToPrint := currentMilestone - lastMilestone
		fmt.Print(strings.Repeat("*", int(starsToPrint)))
		pr.LastStar = pr.BytesRead
	}

	return n, err
}

func downloadZIP(documentURL string, dir string) (string, error) {
	// HTTP-Client
	client := &http.Client{
		Timeout: 360 * time.Second,
	}

	req, err := http.NewRequest("GET", documentURL, nil)
	if err != nil {
		return "", fmt.Errorf("Error setting up request (1): %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:120.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Network error when calling URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected HTTP state code: %s", resp.Status)
	}

	// Try to read proposed filename from response
	fileName := fmt.Sprintf("download-noname-%v.txt", counter)
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if serverFilename, ok := params["filename"]; ok && serverFilename != "" {
				fileName = serverFilename
				fmt.Printf("🎯 filename from header: %s\n", fileName)
				fileName = filepath.Base(fileName)
			}
		}
	}
	fmt.Printf("Downloading file: %v\n", fileName)

	// get URL for download step
	finalURL := resp.Request.URL.String()
	//fmt.Printf("  finalURL: %s\n", finalURL)

	req, err = http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return "", fmt.Errorf("Error setting up request (2): %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:120.0)")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Network error when calling URL: %w", err)
	}
	defer resp.Body.Close()

	progressReader := &ProgressReader{
		Src: resp.Body,
	}

	// read in flat XML bytes
	rawBytes, err := io.ReadAll(progressReader)
	if err != nil {
		return "", fmt.Errorf("Error reading in raw bytes of query result: %v", err)
	}
	fmt.Printf("\nRaw bytes downloaded: %v\n", len(rawBytes))

	// Write file
	fileLocation := fmt.Sprintf("%s/%s", dir, fileName)
	err = os.WriteFile(fileLocation, rawBytes, 0644)
	if err != nil {
		fmt.Printf("❌ Error during writing file: %v\n", err)
		return "", err
	}
	fmt.Printf("💾 File %v successfully downloaded and stored\n", fileLocation)
	counter++

	return fileName, nil
}

func main() {
	// Example document ID. We want TGL 32565.
	tglId := "32565"
	// Download artifact tree for document
	documents := searchTGLSRUDeepParse(tglId)
	os.Mkdir(tglId, 0755)
	// Download all artifacts
	for _, document := range documents {
		fmt.Printf("Processing: %s\n", document)
		zipFileName, err := downloadZIP(document, tglId)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("Processed: %s\n", zipFileName)
	}
}
