package mosspkg

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type matches struct {
	url        string
	match1text string
	match2text string
}

//***************************************************************************
// This function will create a two-dimensional slice containing the
// directories (full path) to send to MOSS for evaluation. The first index addresses
// the specific lab, and the second index addresses the specific student.
// If a student does not have the lab directory, the 2d slice will save the
// directory as an empty string.
// Input: baseDir - the location of the student directories
//		labs - a slice of the labs
// Output: The 2d slice of directories
//		Whether or not the function was successful.
//***************************************************************************
func DirectoryContents(baseDir string, labs []LabInfo) ([][]string, bool) {
	var labsCount int = len(labs)
	var studentCount int

	// Try to read the base directory
	contents, error := ioutil.ReadDir(baseDir)
	if error != nil {
		fmt.Printf("Error reading directory %s: %s\n", baseDir, error)
		return nil, false
	}

	var studentDirs []string

	// Get a list of all the student directories (full path)
	for _, item := range contents {
		if item.IsDir() {
			studentDirs = append(studentDirs, baseDir+"/"+item.Name())
		}
	}
	studentCount = len(studentDirs)

	var studentsLabDirs [][]string = make([][]string, labsCount)
	// For each lab
	for i := range studentsLabDirs {
		studentsLabDirs[i] = make([]string, studentCount)

		// For each student
		for j := range studentsLabDirs[i] {
			tempDir := studentDirs[j] + "/" + labs[i].Name + "/"
			_, error := ioutil.ReadDir(tempDir)
			if error != nil {
				studentsLabDirs[i][j] = ""
			} else {
				studentsLabDirs[i][j] = tempDir
			}
		}
	}

	return studentsLabDirs, true
}

//***************************************************************************
// This function will create MOSS commands to upload the lab files.
// Input: mossDir - the location of the MOSS script
//		studentsLabDirs - a 2d slice of directories
//		labs - a slice of the labs
// 		threshold - ignore matches appearing in at least this many files
// Output: The slice of MOSS commands
//		Whether or not the function was successful.
//***************************************************************************
func CreateMossCommands(mossDir string, studentsLabDirs [][]string, labs []LabInfo, threshold int) ([]string, bool) {
	var commands []string
	var mOption string = "-m " + strconv.Itoa(threshold)

	// For each lab
	for i := range studentsLabDirs {
		var lOption string
		var fileExt []string

		// Set language option and file extenstions
		if labs[i].Language == Golang {
			lOption = "-l java"
			fileExt = append(fileExt, "*.go")
		} else if labs[i].Language == Cpp {
			lOption = "-l cc"
			fileExt = append(fileExt, "*.cpp")
			fileExt = append(fileExt, "*.h")
		} else {
			lOption = "-l java"
			fileExt = append(fileExt, "*.java")
		}

		// Start creating the moss command
		var buf bytes.Buffer
		buf.WriteString(mossDir + "/moss " + lOption + " " + mOption + " -d")

		// For each student
		for j := range studentsLabDirs[i] {

			// If student has the lab
			if studentsLabDirs[i][j] != "" {

				// Add all the files with the appropriate extensions
				for k := range fileExt {
					buf.WriteString(" " + studentsLabDirs[i][j] + fileExt[k])
				}
			}
		}

		buf.WriteString(" > " + labs[i].Name + ".txt &")

		// Add the MOSS command for this lab
		commands = append(commands, buf.String())
	}

	return commands, true
}

//***************************************************************************
// This function saves the data from the specified MOSS URL
// Input: url - the main url for the MOSS results
//		baseDir - where to save the data
//		lab - information about the current lab
// Output: Whether or not the function was successful.
//***************************************************************************
func SaveMossResults(url string, baseDir string, lab LabInfo) bool {
	resultsDir := baseDir + "/" + lab.Name + "/"

	os.MkdirAll(resultsDir, 0764)
	os.Remove(resultsDir + "*.*")

	var comparisons []matches

	// Get web page data
	doc, err := goquery.NewDocument(url)
	if err != nil {
		fmt.Printf("%v\n", err)
		return false
	}

	// Find the table rows
	doc.Find("tr").Each(func(i int, tr *goquery.Selection) {
		var match matches

		// Find the table columns
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			url := ""

			// Find the href attributes
			td.Find("a").Each(func(k int, a *goquery.Selection) {
				url, _ = a.Attr("href")
			})

			// If there is an href attribute
			if url != "" {
				match.url = url
				val := td.Text()
				if match.match1text == "" {
					match.match1text = val
				} else {
					match.match2text = val
				}
			}
		})

		if match.url != "" {
			comparisons = append(comparisons, match)
		}
	})

	// For each URL
	for _, match := range comparisons {
		pos1 := strings.LastIndex(match.url, "/")
		pos2 := strings.LastIndex(match.url, ".html")
		base := match.url[pos1+1 : pos2]
		topFrame := strings.Replace(match.url, ".html", "-top.html", 1)
		leftFrame := strings.Replace(match.url, ".html", "-0.html", 1)
		rightFrame := strings.Replace(match.url, ".html", "-1.html", 1)

		// Get and save web page data
		linkBody, success := GetHtmlData(match.url)
		if !success {
			return false
		}
		ioutil.WriteFile(resultsDir+base+".html", []byte(linkBody), 0644)

		// Get and save top frame
		topBody, success := GetHtmlData(topFrame)
		if !success {
			return false
		}
		ioutil.WriteFile(resultsDir+base+"-top.html", []byte(topBody), 0644)

		// Get and save left frame
		leftBody, success := GetHtmlData(leftFrame)
		if !success {
			return false
		}
		ioutil.WriteFile(resultsDir+base+"-0.html", []byte(leftBody), 0644)

		// Get and save right frame
		rightBody, success := GetHtmlData(rightFrame)
		if !success {
			return false
		}
		ioutil.WriteFile(resultsDir+base+"-1.html", []byte(rightBody), 0644)
	}

	MakeResultsMainPage(resultsDir, lab, comparisons)

	return true
}

//***************************************************************************
// This function returns html data as a string
// Input: url - the location of the html
// Output: The data as a string.
//		Whether or not the function was successful.
//***************************************************************************
func GetHtmlData(url string) (string, bool) {
	var body string = ""
	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		fmt.Printf("%v\n", err)
		return "", false
	} else if resp.StatusCode != 200 {
		fmt.Printf("%v\n", resp.Status)
		return "", false
	} else {
		bodyBuf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%v\n", err)
			return "", false
		}

		body = string(bodyBuf)
	}

	return body, true
}

//***************************************************************************
// This function creates the main html file for the results
// Input: resultsDir - where to save the data
//		lab - information about the current lab
//		comparisons - information about the matches found
// Output: The data as a string.
//		Whether or not the function was successful.
//***************************************************************************
func MakeResultsMainPage(resultsDir string, lab LabInfo, comparisons []matches) {
	var buf bytes.Buffer
	buf.WriteString("<HTML>\n<HEAD>\n<TITLE>")
	buf.WriteString(lab.Name + " Results")
	buf.WriteString("</TITLE>\n</HEAD>\n<BODY>\n")
	buf.WriteString(lab.Name + " Results<br>")
	for _, match := range comparisons {
		pos := strings.LastIndex(match.url, "/")
		base := match.url[pos+1:]

		buf.WriteString("\n<A HREF=\"")
		buf.WriteString(base)
		buf.WriteString("\">")
		buf.WriteString(match.match1text + ", " + match.match2text)
		buf.WriteString("</A><br>")
	}
	buf.WriteString("\n</BODY>\n</HTML>\n")

	ioutil.WriteFile(resultsDir+"results.html", []byte(buf.String()), 0644)
}
