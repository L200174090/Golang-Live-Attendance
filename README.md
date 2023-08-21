# Project Name

Live Attendance System with Go

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Technologies Used](#technologies-used)
- [Installation](#installation)
- [Usage](#usage)
- [Additional Suggestions](#additional-suggestions)
- [Contributing](#contributing)

## Introduction

The Live Attendance System is a robust solution designed to streamline and accurately record attendance data in real-time for employees. Utilizing the power of Go programming, this system offers two core functions: Clock-in and Clock-out, enabling efficient tracking of entry and return times.

## Features

- Clock-in: This function captures the entry time of employees as they begin their workday. It records the precise moment an employee starts their shift.

- Clock-out: This feature records the return time of employees, marking the end of their work session. It ensures accurate tracking of working hours.

- Similarity Check: Upon successful submission of Live Attendance, the system performs a similarity check on the uploaded photo. This comparison is vital to ensure that the photo submitted by the user corresponds closely with the stored image in the database.

- Threshold for Valid Submission: The similarity percentage plays a crucial role in determining the validity of attendance submission. For the submission to be considered successful, the uploaded photo must exhibit a similarity percentage of more than 80% when compared to the stored image. If the similarity falls below this threshold, the system guides the user to either retake the photo or provides a warning that the attendance submission is not considered valid.

## Technologies Used

- HTML, CSS, JavaScript
- Go (Golang)
- PostgreSQL

## Installation

### Database Setup

1. Ensure you have PostgreSQL installed on your system.
2. Navigate to the `databases` folder in the project directory.
3. Using a PostgreSQL tool (e.g., pgAdmin), create a new database.
4. Import the SQL file `attendance.sql` located in the `databases` folder to set up tables and schema.

### Go Application Setup

1. Make sure you have Go installed on your system.
2. Open a terminal and navigate to your project directory.

### Environment Variables (Optional)

1. Create a `.env` file in the project root directory for environment variables (if needed).

## Usage

1. Use the Clock-in and Clock-out features to record attendance.
2. Capture photos for attendance submission.
3. The system will perform similarity checks and provide feedback based on the similarity percentage.

## Run the Application

1. go run main.go

## Access the Application

Open a web browser and navigate to http://localhost:8080 to access the Live Attendance System.

## Additional Suggestions

1. Dockerization (Optional): Dockerize the application for consistent deployment across environments.

2. Authentication and Security (Future Enhancement): Implement user authentication and authorization for data security.

3. Logging and Monitoring (Future Enhancement): Implement logging and monitoring tools for system performance.

4. Unit Testing: Write unit tests for critical parts of the application to ensure reliability.

5. Documentation: Create comprehensive documentation for installation, configuration, and usage.

## Contributing

Contributions are welcome! If you have suggestions or improvements, please open an issue or submit a pull request.
