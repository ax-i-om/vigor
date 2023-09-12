/*
Tempest- Leveraging paste sites as a medium for discovery
Copyright © 2023 ax-i-om <addressaxiom@pm.me>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package mega contains functions that can be used to accurately extract and validate Mega links.
package mega

import (
	"io"
	"regexp"
	"strings"

	"github.com/ax-i-om/tempest/internal/hdl"
	"github.com/ax-i-om/tempest/internal/models"
	"github.com/ax-i-om/tempest/internal/req"
)

// RegEx expressions for extracting metadata
// Size RegEx expression
var size *regexp.Regexp = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([KMGTP]?B)`)

// Extract header property that contains the file count/content
var filesLine *regexp.Regexp = regexp.MustCompile(`<meta property="og:description" content="(.*?)" />`)

// Extract the only number in said string (file count/content)
var digits *regexp.Regexp = regexp.MustCompile(`\d+`)

// Compile the RegEx expression to be used in the identification and extraction of the Mega links
var mLink *regexp.Regexp = regexp.MustCompile("(https|http)://mega.nz/(folder|file)/([a-zA-Z0-9]{0,8})#([a-zA-Z0-9_-]{43}|[a-zA-Z0-9_-]{22})")

// Compile the RegEx expression to be used in the extraction of the ID
var mID *regexp.Regexp = regexp.MustCompile("([a-zA-Z0-9]{8}#)")

// Extract returns a slice of all Mega links contained within a string, if any.
func Extract(res string) ([]string, error) {
	// Return all Mega links found within an http response
	return mLink.FindAllString(res, -1), nil
}

// Validate takes a Mega link/URL and passes it to the Mega API to check whether or not it is online.
func Validate(x string) (bool, error) {
	// Extracts the eight-character identifier from the URL using the pound (#) symbol as context.
	pre := mID.FindString(x)
	// Remove the pound (#) symbol from the extracted ID
	post := strings.ReplaceAll(pre, "#", "")

	// Append the ID (post) to the URL
	url := "https://g.api.mega.co.nz/cs?id=5644474&n=" + post

	// Perform a GET request using the pre-formatted URL
	res, err := req.GetRes(url)
	if err != nil {
		return false, err
	}

	// Prepare the contents of the response to be read
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	// Read the response, if the body response contains a -2, then the Mega link is valid. If it contains a -9 or anything else, it is invalid.
	if strings.Contains(string(body), "-2") {
		return true, res.Body.Close()
	} else {
		return false, res.Body.Close()
	}
}

// Delegate takes a string as an argument and returns a slice of valid Mega links found within the response (if any) and an error
func Delegate(res string) ([]models.Entry, error) {
	// Use Extract() to extract any existing Mega links from the response
	x, err := Extract(res)
	if err != nil {
		return nil, err
	}
	// Check if the return slice of Mega links is empty
	if len(x) > 0 {
		// Create a new, empty slice where we will append any valid Mega links
		var results []models.Entry = nil
		// Loop through each Mega link within the slice
		for _, v := range x {
			// Call the Validate function in order to check whether or not the link is valid
			x, err := Validate(v)
			if err != nil {
				// If any error occurs during the validation process, stop the current iteration and immediately begin with the next link within the slice
				continue
			}

			// If x, the bool returned by Validate(), is true: append the link to the specified results slice.
			if x {
				// Get body contents of the mega folder/file
				res, err := req.GetRes(v)
				if err != nil {
					continue
				}

				// Read results of the *http.Response body
				body, err := io.ReadAll(res.Body)
				if err != nil {
					continue
				}

				// Convert read results to a string
				contents := string(body)

				// Initialize variable to potentially be accessed in the following conditional
				count := "N/A"
				fTyp := "N/A"
				// If the link contains the word "folder", then the mega link type is that of a folder.
				if strings.Contains(v, "folder") {
					// Extract file count header
					fl := filesLine.FindString(contents)
					// Extract number from header
					count = digits.FindString(fl)
					// Set fTyp to Folder
					fTyp = "Folder"
				} else { // (handle file link here)
					fTyp = "File"
				}

				// Extract the string that specifies the cumulative size of the mega folder contents or the size of a mega file
				ss := size.FindString(contents)

				// Create type Entry and specify the respective values
				ent := models.Entry{Link: v, Service: "Mega", Type: fTyp, Size: ss, FileCount: count, LastValidation: hdl.Time()}
				// Append the entry to the results slice to be returned to the main runner
				results = append(results, ent)
			}
		}
		// When the loop is finished, return the results slice
		return results, nil
	}
	// Return nothing, if nothing happens (bruh)
	return nil, nil
}
