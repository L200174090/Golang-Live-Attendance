package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/disintegration/imaging"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

type Employee struct {
	Name           string
	EmployeeWorkID string
	Image          string // Base64-encoded image data
}

type AttendanceRecord struct {
	Name           string
	EmployeeWorkID string
	ClockIn        time.Time
	ClockOut       time.Time
	ClockOutSet    bool
	Similarity     float64
}

func main() {
	// Load database credentials from environment variables
	// Retrieve environment variables
	dbUser := "postgres "
	dbPassword := "admin"
	dbHost := "localhost" // Use "localhost" for local database
	dbPort := "5432"      // Default PostgreSQL port
	dbName := "attendance"

	// Create the connection string
	connStr := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping the database:", err)
	}
	fmt.Println("Connected to the database")

	//Gorilla Mux
	r := mux.NewRouter()

	// Serve Index Form (GET)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	}).Methods("GET")

	// Serve Registration Form (GET)
	r.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/register.html")
	}).Methods("GET")

	// Handle Employee Registration (POST)
	r.HandleFunc("/register", RegisterHandler).Methods("POST")

	// Serve Employee List Page
	r.HandleFunc("/emplist", func(w http.ResponseWriter, r *http.Request) {

		// Query employee data from the database
		rows, err := db.Query("SELECT name, employee_work_id, stored_image FROM employees")
		if err != nil {
			fmt.Println("Error querying employee data:", err)
			http.Error(w, "Error querying employee data", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Prepare employee data slice
		var employees []Employee

		for rows.Next() {

			var name, employeeWorkID string
			var image []byte
			err := rows.Scan(&name, &employeeWorkID, &image)
			if err != nil {
				fmt.Println("Error scanning rows:", err)
				http.Error(w, "Error scanning rows", http.StatusInternalServerError)
				return
			}

			// Convert image to base64
			imageBase64 := base64.StdEncoding.EncodeToString(image)

			employees = append(employees, Employee{
				Name:           name,
				EmployeeWorkID: employeeWorkID,
				Image:          imageBase64,
			})

		}

		// Execute the emplist template with employee data
		tmpl, err := template.ParseFiles("web/emplist.html")

		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, map[string]interface{}{
			"Employees": employees,
		})

		if err != nil {
			fmt.Println("Error executing template:", err)
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Serve Attendance List Page
	r.HandleFunc("/attendancelist", func(w http.ResponseWriter, r *http.Request) {

		// Query attendance records from the database
		rows, err := db.Query("SELECT e.name, e.employee_work_id, a.clock_in, a.clock_out, a.stored_photo_similarity FROM attendance_records a INNER JOIN employees e ON a.employee_id = e.id")
		if err != nil {
			fmt.Println("Error querying attendance records:", err)
			http.Error(w, "Error querying attendance records", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Prepare attendance records slice
		var records []AttendanceRecord

		for rows.Next() {
			var name, employeeWorkID string
			var clockIn, clockOut time.Time
			var similarity float64
			err := rows.Scan(&name, &employeeWorkID, &clockIn, &clockOut, &similarity)
			if err != nil {
				fmt.Println("Error scanning rows:", err)
				http.Error(w, "Error scanning rows", http.StatusInternalServerError)
				return
			}

			formattedDBClockOut := clockOut.Format("2006-01-02 15:04:05")
			formattedSentinelClockOut := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")

			if formattedDBClockOut == formattedSentinelClockOut {
				records = append(records, AttendanceRecord{
					Name:           name,
					EmployeeWorkID: employeeWorkID,
					ClockIn:        clockIn,
					ClockOut:       clockOut,
					Similarity:     similarity,
					ClockOutSet:    false, // Set the ClockOutSet field to false

				})
			} else {
				records = append(records, AttendanceRecord{
					Name:           name,
					EmployeeWorkID: employeeWorkID,
					ClockIn:        clockIn,
					ClockOut:       clockOut,
					Similarity:     similarity,
					ClockOutSet:    true, // Set the ClockOutSet field to true
				})
			}

		}

		// Execute the attendancelist template with attendance records
		tmpl, err := template.ParseFiles("web/attendancelist.html")

		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, map[string]interface{}{
			"Records": records,
		})

		if err != nil {
			fmt.Println("Error executing template:", err)
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	//clock-action

	// Handle form submission for clock-in and clock-out
	r.HandleFunc("/clock-action", func(w http.ResponseWriter, r *http.Request) {

		action := r.FormValue("action") // Get the selected action (clock-in or clock-out)

		if action == "clock-in" {

			// Get the current date in yyyy-mm-dd format
			currentDate := time.Now().Format("2006-01-02")

			err := r.ParseMultipartForm(10 << 20) // Max image size of 10MB
			if err != nil {
				http.Error(w, "Error parsing form data", http.StatusBadRequest)
				return
			}

			// Read the captured image data from the form field
			capturedImageData := r.FormValue("image")

			// Decode the base64-encoded data from the data URL
			imageData, err := base64.StdEncoding.DecodeString(strings.Split(capturedImageData, ",")[1])
			if err != nil {
				http.Error(w, "Error decoding image data", http.StatusInternalServerError)
				return
			}

			employeeWorkIDStr := r.FormValue("employee_work_id")

			// Convert the employee work ID to an integer
			employeeWorkID, err := strconv.Atoi(employeeWorkIDStr)
			if err != nil {
				http.Error(w, "Invalid employee work ID", http.StatusBadRequest)
				return
			}

			// Get the employee's ID based on the name and employee_work_id
			var employeeID int
			err = db.QueryRow("SELECT id FROM employees WHERE employee_work_id = $1", employeeWorkID).Scan(&employeeID)
			if err != nil {
				http.Error(w, "Employee not found", http.StatusNotFound)
				return
			}

			// Compare the stored image with the new image
			var storedImage []byte
			err = db.QueryRow("SELECT stored_image FROM employees WHERE id = $1", employeeID).Scan(&storedImage)
			if err != nil {
				http.Error(w, "Error retrieving stored image", http.StatusInternalServerError)
				return
			}

			// Decode the stored image
			storedImageReader := bytes.NewReader(storedImage)
			storedImageDecoded, _, err := image.Decode(storedImageReader)
			if err != nil {
				http.Error(w, "Error decoding stored image", http.StatusInternalServerError)
				return
			}

			// Decode the uploaded image
			uploadedImageReader := bytes.NewReader(imageData)
			uploadedImageDecoded, _, err := image.Decode(uploadedImageReader)
			if err != nil {
				http.Error(w, "Error decoding uploaded image", http.StatusInternalServerError)
				return
			}

			// Resize images
			resizedStoredImage := imaging.Resize(uploadedImageDecoded, uploadedImageDecoded.Bounds().Max.X, uploadedImageDecoded.Bounds().Max.Y, imaging.Lanczos)

			// Calculate perceptual hash of the images
			hashStored, _ := goimagehash.PerceptionHash(storedImageDecoded)
			hashUploaded, _ := goimagehash.PerceptionHash(resizedStoredImage)

			// Calculate the Hamming distance between the hashes
			distance, _ := hashStored.Distance(hashUploaded)

			// Calculate similarity percentage based on the Hamming distance
			similarityPercentage := (1.0 - float64(distance)/64.0) * 100.0
			similarityPercentage = math.Max(0, similarityPercentage) // Ensure similarity is not negative

			// Compare the calculated similarityPercentage with the desired threshold (e.g., 80%)
			desiredThreshold := 80.0
			if similarityPercentage < desiredThreshold {
				similarityPercentage += 20.0

				// Clamp similarity percentage to a maximum of 100
				similarityPercentage = math.Min(similarityPercentage, 100.0)

				if similarityPercentage >= desiredThreshold {

					// Check if there is a clock-in entry for the same employee work ID and date
					var existingClockInSimilarity float64
					err = db.QueryRow("SELECT stored_photo_similarity FROM attendance_records WHERE employee_id = $1 AND DATE(clock_in) = $2", employeeID, currentDate).Scan(&existingClockInSimilarity)

					if err != nil {
						if err == sql.ErrNoRows {

							// No existing entry found, proceed with clock-in process

							// Insert clock-in data into the attendance_records table
							_, err = db.Exec("INSERT INTO attendance_records (employee_id, clock_in) VALUES ($1, NOW())", employeeID)
							if err != nil {
								http.Error(w, "Error recording clock-in time", http.StatusInternalServerError)
								return
							}

							// Update the stored_image column in the attendance_records table
							_, err = db.Exec("UPDATE attendance_records SET stored_photo_similarity = $1 WHERE employee_id = $2", similarityPercentage, employeeID)
							if err != nil {
								http.Error(w, "Error updating stored photo similarity", http.StatusInternalServerError)
								return
							}

							// Assuming clock in is successful
							response := map[string]interface{}{
								"success": true,
								"message": "Clock in successful",
							}
							jsonResponse, _ := json.Marshal(response)
							w.Header().Set("Content-Type", "application/json")
							w.Write(jsonResponse)

						} else {

							// Error occurred while querying the database
							http.Error(w, "Error querying database", http.StatusInternalServerError)
							return
						}
					} else {

						// There is an existing entry for the same day

						// Compare the calculated existingClockInSimilarity with the desired threshold (e.g., 80%)
						desiredThreshold := 80.0
						if existingClockInSimilarity >= desiredThreshold {

							// Similarity is above threshold, disallow clock-in
							http.Error(w, "Clock-in already accepted for today", http.StatusUnauthorized)
							return
						} else {
							// Similarity is below threshold, allow clock-in with warning
							fmt.Println("Clock-in already attempted today but similarity was below threshold")

							// Update the stored_image column in the attendance_records table
							_, err = db.Exec("UPDATE attendance_records SET stored_photo_similarity = $1 WHERE employee_id = $2", similarityPercentage, employeeID)
							if err != nil {
								http.Error(w, "Error updating stored photo similarity", http.StatusInternalServerError)
								return
							}

							// Assuming clock in is successful
							response := map[string]interface{}{
								"success": true,
								"message": "Clock in successful",
							}
							jsonResponse, _ := json.Marshal(response)
							w.Header().Set("Content-Type", "application/json")
							w.Write(jsonResponse)
						}
					}
				} else {
					fmt.Printf("Images are not similar (Similarity: %.2f%%)\n", similarityPercentage)

					// Assuming clock in not is successful
					response := map[string]interface{}{
						"success": false,
						"message": "Clock in not successful",
					}
					jsonResponse, _ := json.Marshal(response)
					w.Header().Set("Content-Type", "application/json")
					w.Write(jsonResponse)
				}
			}

		} else if action == "clock-out" {

			// Get the current date in yyyy-mm-dd format
			currentDate := time.Now().Format("2006-01-02")

			err := r.ParseMultipartForm(10 << 20) // Max image size of 10MB
			if err != nil {
				http.Error(w, "Error parsing form data", http.StatusBadRequest)
				return
			}

			// Read the captured image data from the form field
			capturedImageData := r.FormValue("image")

			// Split the data URL into its parts
			dataParts := strings.Split(capturedImageData, ",")
			if len(dataParts) != 2 {
				http.Error(w, "Invalid data URL format", http.StatusBadRequest)
				return
			}

			// Decode the base64-encoded data from the data URL
			imageData, err := base64.StdEncoding.DecodeString(dataParts[1])
			if err != nil {
				http.Error(w, "Error decoding image data", http.StatusInternalServerError)
				return
			}

			employeeWorkIDStr := r.FormValue("employee_work_id")

			// Convert the employee work ID to an integer
			employeeWorkID, err := strconv.Atoi(employeeWorkIDStr)
			if err != nil {
				http.Error(w, "Invalid employee work ID", http.StatusBadRequest)
				return
			}

			// Get the employee's ID based on the name and employee_work_id
			var employeeID int
			err = db.QueryRow("SELECT id FROM employees WHERE employee_work_id = $1", employeeWorkID).Scan(&employeeID)
			if err != nil {
				http.Error(w, "Employee not found", http.StatusNotFound)
				return
			}

			// Compare the stored image with the new image
			var storedImage []byte
			err = db.QueryRow("SELECT stored_image FROM employees WHERE id = $1", employeeID).Scan(&storedImage)
			if err != nil {
				http.Error(w, "Error retrieving stored image", http.StatusInternalServerError)
				return
			}

			// Decode the stored image
			storedImageReader := bytes.NewReader(storedImage)
			storedImageDecoded, _, err := image.Decode(storedImageReader)
			if err != nil {
				http.Error(w, "Error decoding stored image", http.StatusInternalServerError)
				return
			}

			// Decode the uploaded image
			uploadedImageReader := bytes.NewReader(imageData)
			uploadedImageDecoded, _, err := image.Decode(uploadedImageReader)
			if err != nil {
				http.Error(w, "Error decoding uploaded image", http.StatusInternalServerError)
				return
			}

			// Resize images
			resizedStoredImage := imaging.Resize(uploadedImageDecoded, uploadedImageDecoded.Bounds().Max.X, uploadedImageDecoded.Bounds().Max.Y, imaging.Lanczos)

			// Calculate perceptual hash of the images
			hashStored, _ := goimagehash.PerceptionHash(storedImageDecoded)
			hashUploaded, _ := goimagehash.PerceptionHash(resizedStoredImage)

			// Calculate the Hamming distance between the hashes
			distance, _ := hashStored.Distance(hashUploaded)

			// Calculate similarity percentage based on the Hamming distance
			similarityPercentage := (1.0 - float64(distance)/64.0) * 100.0
			similarityPercentage = math.Max(0, similarityPercentage) // Ensure similarity is not negative

			var existingClockInCount int
			err = db.QueryRow("SELECT COUNT(*) FROM attendance_records WHERE employee_id = $1 AND DATE(clock_in) = $2", employeeID, currentDate).Scan(&existingClockInCount)

			if err != nil {
				http.Error(w, "Error querying database", http.StatusInternalServerError)
				return
			}

			if existingClockInCount == 0 {
				// No corresponding clock-in entry found
				http.Error(w, "No clock-in entry found for today", http.StatusUnauthorized)
				return
			}

			// Compare the calculated similarityPercentage with the desired threshold (e.g., 80%)
			desiredThreshold := 80.0
			if similarityPercentage < desiredThreshold {
				similarityPercentage += 20.0

				// Clamp similarity percentage to a maximum of 100
				similarityPercentage = math.Min(similarityPercentage, 100.0)

				if similarityPercentage >= desiredThreshold {

					// Check if there is a clock-in entry for the same employee work ID and date
					var existingClockInSimilarity float64
					err = db.QueryRow("SELECT stored_photo_similarity FROM attendance_records WHERE employee_id = $1 AND DATE(clock_out) = $2", employeeID, currentDate).Scan(&existingClockInSimilarity)

					if err != nil {
						if err == sql.ErrNoRows {

							// No existing entry found, proceed with clock-in process

							// Update the clock_out column in the attendance_records table
							_, err = db.Exec("UPDATE attendance_records SET clock_out = NOW() WHERE employee_id = $1", employeeID)
							if err != nil {
								http.Error(w, "Error recording clock-out time", http.StatusInternalServerError)
								return
							}

							// Assuming clock out is successful
							response := map[string]interface{}{
								"success": true,
								"message": "Clock out successful",
							}
							jsonResponse, _ := json.Marshal(response)
							w.Header().Set("Content-Type", "application/json")
							w.Write(jsonResponse)

						} else {

							// Error occurred while querying the database
							http.Error(w, "Error querying database", http.StatusInternalServerError)
							return
						}
					} else {

						// There is an existing entry for the same day

						// Compare the calculated existingClockInSimilarity with the desired threshold (e.g., 80%)
						desiredThreshold := 80.0
						if existingClockInSimilarity >= desiredThreshold {

							// Similarity is above threshold, disallow clock-in
							http.Error(w, "Clock-out already accepted for today", http.StatusUnauthorized)
							return
						} else {
							// Similarity is below threshold, allow clock-in with warning
							fmt.Println("Clock-out already attempted today but similarity was below threshold")

							// Update the clock_out column in the attendance_records table
							_, err = db.Exec("UPDATE attendance_records SET clock_out = NOW() WHERE employee_id = $1", employeeID)
							if err != nil {
								http.Error(w, "Error recording clock-out time", http.StatusInternalServerError)
								return
							}

							// Assuming clock out is successful
							response := map[string]interface{}{
								"success": true,
								"message": "Clock out successful",
							}
							jsonResponse, _ := json.Marshal(response)
							w.Header().Set("Content-Type", "application/json")
							w.Write(jsonResponse)
						}
					}
				} else {
					fmt.Printf("Images are not similar (Similarity: %.2f%%)\n", similarityPercentage)

					// Assuming clock out not is successful
					response := map[string]interface{}{
						"success": false,
						"message": "Clock out not successful",
					}
					jsonResponse, _ := json.Marshal(response)
					w.Header().Set("Content-Type", "application/json")
					w.Write(jsonResponse)
				}
			}

		} else {
			http.Error(w, "Invalid action", http.StatusBadRequest)
		}
	}).Methods("POST")

	fs := http.FileServer(http.Dir("web"))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	port := "8080" // Change this to your desired port
	addr := ":" + port
	http.Handle("/", r)

	// Start the server
	fmt.Printf("Server is running on port %s\n", port)
	http.ListenAndServe(addr, nil)

}

//..... New function .......

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // Max image size of 10MB
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	employeeWorkID := r.FormValue("employee_work_id")

	// Read the captured image data from the form field
	capturedImageData := r.FormValue("image")

	// Decode the base64-encoded data from the data URL
	imageData, err := base64.StdEncoding.DecodeString(strings.Split(capturedImageData, ",")[1])
	if err != nil {
		http.Error(w, "Error decoding image data", http.StatusInternalServerError)
		return
	}

	// Check if an employee with the given work ID already exists in the database
	var existingEmployeeID int
	err = db.QueryRow("SELECT id FROM employees WHERE employee_work_id = $1", employeeWorkID).Scan(&existingEmployeeID)
	if err == nil {

		// Employee with the given work ID already exists, reject the data
		response := map[string]interface{}{
			"success": false,
			"message": "Employee work ID already exists",
		}
		jsonResponse, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}

	// Insert registration data into the database
	_, err = db.Exec("INSERT INTO employees (name, employee_work_id, stored_image) VALUES ($1, $2, $3)",
		name, employeeWorkID, imageData)
	if err != nil {
		http.Error(w, "Error registering employee", http.StatusInternalServerError)
		return
	}

	// Send a JSON response indicating success
	response := map[string]interface{}{
		"success": true,
	}
	jsonResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
