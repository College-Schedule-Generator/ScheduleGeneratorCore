package main

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"main/db"
	"strconv"
	"strings"
)

const SCHOOL_ID string = "2649"

type Time struct {
	Hour   int `json:"Hour"`
	Minute int `json:"Minute"`
}

type TimeRange struct {
	StartTime Time
	EndTime   Time
}

type DateRange struct {
	StartMonth int `json:"StartMonth"`
	StartDay   int `json:"StartDay"`
	EndMonth   int `json:"EndMonth"`
	EndDay     int `json:"EndDay"`
}

type MeetingTime struct {
	Monday    bool `json:"Monday"`
	Tuesday   bool `json:"Tuesday"`
	Wednesday bool `json:"Wednesday"`
	Thursday  bool `json:"Thursday"`
	Friday    bool `json:"Friday"`
	Saturday  bool `json:"Saturday"`
	Sunday    bool `json:"Sunday"`
	StartTime Time `json:"StartTime"`
	EndTime   Time `json:"EndTime"`
}

type Class struct {
	CourseName          string        `json:"courseName"`
	ClassID             string        `json:"classID"`
	Instructor          string        `json:"instructor"`
	Availability        string        `json:"availability"`
	InstructionalMethod string        `json:"instructionalMethod"`
	MeetingTimes        []MeetingTime `json:"meetingTimes"`
	Date                DateRange     `json:"date"`
}

type ClassEnhanced struct {
	CourseName          string        `json:"courseName"`
	ClassID             string        `json:"classID"`
	Instructor          string        `json:"instructor"`
	InstructorRating    float32       `json:"instructorRating"`
	Availability        string        `json:"availability"`
	InstructionalMethod string        `json:"instructionalMethod"`
	MeetingTimes        []MeetingTime `json:"meetingTimes"`
	Date                DateRange     `json:"date"`
}

type School struct {
	Timestamp int64   `json:"timestamp"`
	School    string  `json:"school"`
	Classes   []Class `json:"classes"`
}

type ProfessorType struct {
	Department      string `json:"department"`
	SchoolID        string `json:"schoolId"`
	InstitutionName string `json:"institutionName"`
	FirstName       string `json:"firstName"`
	MiddleName      string `json:"middleName"`
	LastName        string `json:"lastName"`
	Id              int    `json:"id"`
	TotalRatings    int    `json:"totalRatings"`
	RatingsClass    string `json:"ratingsClass"`
	ContentType     string `json:"contentType"`
	CategoryType    string `json:"categoryType"`
	OverallRating   string `json:"overallRating"`
}

type ProfessorExport struct {
	Timestamp  int64           `json:"timestamp"`
	SchoolId   string          `json:"schoolId"`
	Professors []ProfessorType `json:"professors"`
}

type UserScheduleConstraints struct {
	schoolId string
	courses  []string

	mondayTime    []TimeRange
	tuesdayTime   []TimeRange
	wednesdayTime []TimeRange
	thursdayTime  []TimeRange
	fridayTime    []TimeRange
	saturdayTime  []TimeRange
	sundayTime    []TimeRange

	instructionalMethods []string
	availability         []string
}

func main() {
	// given a list of courses and conditions (test/example data)
	courses := []string{"MATH 005B", "PHYS 008A", "ENGL 001A"}
	instructionalMethods := []string{"IP", "HY"}
	availability := []string{"open", "waitlisted"}

	startTime := Time{8, 0}
	endTime := Time{23, 0}
	timeRange := TimeRange{startTime, endTime}
	mondayTime := []TimeRange{timeRange}

	// This is the algorithms INPUT
	userScheduleConstraints := UserScheduleConstraints{schoolId: SCHOOL_ID, courses: courses, instructionalMethods: instructionalMethods, availability: availability,
		mondayTime: mondayTime, tuesdayTime: mondayTime, wednesdayTime: mondayTime, thursdayTime: mondayTime}

	// pull schedule data from db
	school, err := fetchClassData(userScheduleConstraints.schoolId)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// List of all courses specified in "courses" constraint
	classes := filterCourses(school, userScheduleConstraints)

	// Remove courses that do not fit schedule / instructionalMethods / availability
	classes = getClassesThatFitScheduleConstraints(classes, userScheduleConstraints)

	// Fetch professor rating from database
	professorsExport, err := fetchProfessorData(SCHOOL_ID)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Integrate rate my professor ratings into classes data
	enhancedClasses := []ClassEnhanced{}
	enhancedClasses = integrateRatingsIntoClassData(classes, professorsExport.Professors)

	// algorithm
	resultClasses := []ClassEnhanced{}
	resultClasses = generateSchedule(enhancedClasses)

	// output result -> return to sender
	fmt.Println(len(resultClasses))
}

func generateSchedule(classes []ClassEnhanced) []ClassEnhanced {
	return []ClassEnhanced{}
}

// Filters the courses we want and returns them
func filterCourses(school School, userScheduleConstraints UserScheduleConstraints) []Class {
	classes := []Class{}

	for _, class := range school.Classes {
		for _, courseName := range userScheduleConstraints.courses {
			if courseName == class.CourseName {
				classes = append(classes, class)
			}
		}
	}

	return classes
}

// TODO: Improve accuracy of this function. It still has trouble matching some of the RMF names to the schedule names
func integrateRatingsIntoClassData(classes []Class, professors []ProfessorType) []ClassEnhanced {
	enhancedClasses := []ClassEnhanced{}
	found := false

	for _, class := range classes {
		found = false
		for _, professor := range professors {
			if (strings.Contains(class.Instructor, strings.TrimSpace(strings.ToLower(professor.FirstName))) || len(professor.FirstName) == 0) &&
				(strings.Contains(class.Instructor, strings.TrimSpace(strings.ToLower(professor.LastName))) || len(professor.LastName) == 0) {
				rating, err := strconv.ParseFloat(professor.OverallRating, 32)
				if err != nil {
					break
				}

				enhancedClasses = append(enhancedClasses, ClassEnhanced{class.CourseName, class.ClassID,
					class.Instructor, float32(rating), class.Availability,
					class.InstructionalMethod, class.MeetingTimes, class.Date})

				found = true
				break
			}
		}

		if !found {
			enhancedClasses = append(enhancedClasses, ClassEnhanced{class.CourseName, class.ClassID,
				class.Instructor, -1, class.Availability,
				class.InstructionalMethod, class.MeetingTimes, class.Date})
		}
	}

	return enhancedClasses
}

func getClassesThatFitScheduleConstraints(classes []Class, constraints UserScheduleConstraints) []Class {
	newClasses := []Class{}

	fitsSchedule := false

	meetingTimeStart := 0
	meetingTimeEnd := 0
	scheduleTimeStart := 0
	scheduleTimeEnd := 0

	available := func(meetingTime MeetingTime, mondayTimeRange []TimeRange) bool {
		for _, constraintTime := range mondayTimeRange {
			meetingTimeStart = (meetingTime.StartTime.Hour * 100) + meetingTime.StartTime.Minute
			meetingTimeEnd = (meetingTime.EndTime.Hour * 100) + meetingTime.EndTime.Minute

			scheduleTimeStart = (constraintTime.StartTime.Hour * 100) + constraintTime.StartTime.Minute
			scheduleTimeEnd = (constraintTime.EndTime.Hour * 100) + constraintTime.EndTime.Minute

			// check if meeting range is in schedule range
			if ((meetingTimeStart - scheduleTimeStart) >= 0) &&
				(meetingTimeEnd-scheduleTimeEnd <= 0) {
				// It fits in our schedule
				return true
			}
		}

		return false
	}

	for _, class := range classes {
		fitsSchedule = false

		// Instructional Method
		for _, preferredInstruction := range constraints.instructionalMethods {
			if preferredInstruction == class.InstructionalMethod {
				fitsSchedule = true
			}
		}
		if !fitsSchedule {
			continue
		}
		fitsSchedule = false

		// Availability
		for _, preferredAvailability := range constraints.availability {
			if preferredAvailability == class.Availability {
				fitsSchedule = true
			}
		}
		if !fitsSchedule {
			continue
		}
		fitsSchedule = false

		// if class has no meeting times
		if len(class.MeetingTimes) == 0 {
			fitsSchedule = true
		} else {
			// Looping through each class meeting time
			for _, meetingTime := range class.MeetingTimes {
				// MONDAY
				if meetingTime.Monday {
					if available(meetingTime, constraints.mondayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// TUESDAY
				if meetingTime.Tuesday {
					if available(meetingTime, constraints.tuesdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// WEDNESDAY
				if meetingTime.Wednesday {
					if available(meetingTime, constraints.wednesdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// THURSDAY
				if meetingTime.Thursday {
					if available(meetingTime, constraints.thursdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// FRIDAY
				if meetingTime.Friday {
					if available(meetingTime, constraints.fridayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// SATURDAY
				if meetingTime.Saturday {
					if available(meetingTime, constraints.saturdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// SUNDAY
				if meetingTime.Sunday {
					if available(meetingTime, constraints.sundayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}
			}
		}

		if fitsSchedule {
			newClasses = append(newClasses, class)
		}
	}

	return newClasses
}

// TODO: This will be fine for the first prototype, but we need to cache data and not fetch it everytime
// Fetches most recent class data from MongoDB
func fetchClassData(schoolId string) (School, error) {
	// Get collection from database
	collection, err := db.GetDBCollection("Classes")

	if err != nil {
		fmt.Println(err)
		return School{}, errors.New("unable to fetch collection from database")
	}

	// Search for specified course in database
	opts := options.FindOne().SetSort(bson.M{"$natural": -1}) // starts searching from most recent documents
	cursor := collection.FindOne(context.TODO(), bson.M{"schoolid": schoolId}, opts)

	// Deserialize result
	var elem School
	err = cursor.Decode(&elem)

	if err != nil {
		return School{}, errors.New("did not find specified document")
	}

	return elem, nil
}

func fetchProfessorData(schoolId string) (ProfessorExport, error) {
	// Get collection from database
	collection, err := db.GetDBCollection("Professors")

	if err != nil {
		fmt.Println(err)
		return ProfessorExport{}, errors.New("unable to fetch collection from database")
	}

	// Search for specified course in database
	opts := options.FindOne().SetSort(bson.M{"$natural": -1}) // starts searching from most recent documents
	cursor := collection.FindOne(context.TODO(), bson.M{"schoolid": schoolId}, opts)

	// Deserialize result
	var elem ProfessorExport
	err = cursor.Decode(&elem)

	if err != nil {
		return ProfessorExport{}, errors.New("did not find specified document")
	}

	return elem, nil
}
