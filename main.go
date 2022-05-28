package main

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"main/db"
	"sort"
	"strconv"
	"strings"
)

const SCHOOL_ID string = "2649" // PCC school id FROM rate my professor

type Time struct {
	Hour   int `json:"Hour"`
	Minute int `json:"Minute"`
}

type TimeRange struct {
	StartTime Time `json:"startTime"`
	EndTime   Time `json:"endTime"`
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
	SchoolId string   `json:"schoolId"`
	Courses  []string `json:"courses"`

	MondayTime    []TimeRange `json:"mondayTime"`
	TuesdayTime   []TimeRange `json:"tuesdayTime"`
	WednesdayTime []TimeRange `json:"wednesdayTime"`
	ThursdayTime  []TimeRange `json:"thursdayTime"`
	FridayTime    []TimeRange `json:"fridayTime"`
	SaturdayTime  []TimeRange `json:"saturdayTime"`
	SundayTime    []TimeRange `json:"sundayTime"`

	InstructionalMethods []string `json:"instructionalMethods"`
	Availability         []string `json:"availability"`
}

func main() {
	// given a list of courses and conditions (THIS IS TEST DATA)
	courses := []string{"MATH 008", "MATH 003", "PHIL 025", "SOC 001", "MATH 005A", "CHEM 001A"}
	instructionalMethods := []string{"HY", "FO", "IP"}
	availability := []string{"open", "waitlisted", "closed"}

	startTime := Time{8, 30}
	endTime := Time{12, 0}
	timeRange := TimeRange{startTime, endTime}

	startTime1 := Time{13, 0}
	endTime1 := Time{20, 45}
	timeRange1 := TimeRange{startTime1, endTime1}

	mondayTime := []TimeRange{timeRange, timeRange1}

	// This is the algorithms INPUT (TEST DATA)
	userScheduleConstraints := UserScheduleConstraints{SchoolId: SCHOOL_ID, Courses: courses, InstructionalMethods: instructionalMethods, Availability: availability,
		MondayTime: mondayTime, TuesdayTime: mondayTime, WednesdayTime: mondayTime, ThursdayTime: mondayTime}

	// pull schedule data from db
	school, err := fetchClassData(userScheduleConstraints.SchoolId)
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

	// USEFUL REPORTING INFO
	//for _, class := range enhancedClasses {
	//	fmt.Print("Class ID: " + class.ClassID)
	//	fmt.Print(" Instructor: " + class.Instructor)
	//	fmt.Print(" Rating: ")
	//	fmt.Print(class.InstructorRating)
	//	fmt.Println()
	//}

	// algorithm
	resultClasses := []ClassEnhanced{}
	resultClasses = generateSchedule(enhancedClasses)

	// output result -> return to sender
	fmt.Println(len(resultClasses))
}

func generateSchedule(classes []ClassEnhanced) []ClassEnhanced {
	type TempClass struct {
		courseName  string
		occurrences int
		classes     []ClassEnhanced
	}

	tempClasses := []TempClass{}
	found := false

	// Count how many classes of each course we have
	for _, class := range classes {
		found = false

		for i, tempClass := range tempClasses {
			if tempClass.courseName == class.CourseName {
				tempClasses[i].occurrences++
				tempClasses[i].classes = append(tempClasses[i].classes, class)
				found = true
				break
			}
		}

		if !found {
			tempClasses = append(tempClasses, TempClass{class.CourseName, 1, []ClassEnhanced{class}})
		}
	}

	largestCol := 0

	for _, tempClass := range tempClasses {
		// USEFUL REPORTING INFO
		//fmt.Print("Course Name: " + tempClass.courseName)
		//fmt.Print(" Occurrences: " + strconv.Itoa(int(tempClass.occurrences)))
		//fmt.Println()
		if tempClass.occurrences > largestCol {
			largestCol = tempClass.occurrences
		}
	}

	// Main Algorithm (brute force method, a more efficient method would be better!)
	type Schedule struct {
		classes []ClassEnhanced
		rating  float32
	}

	possibleSchedules := []Schedule{}

	for _, col1 := range tempClasses[0].classes {
		if len(tempClasses) == 1 {
			// we stop here
			possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1}, col1.InstructorRating})
			continue
		}

		for _, col2 := range tempClasses[1].classes {
			if len(tempClasses) == 2 {
				// we stop here
				if isScheduleValid([]ClassEnhanced{col1, col2}) {
					possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1, col2}, col1.InstructorRating + col2.InstructorRating})
				}
				continue
			}

			for _, col3 := range tempClasses[2].classes {
				if len(tempClasses) == 3 {
					// we stop here
					if isScheduleValid([]ClassEnhanced{col1, col2, col3}) {
						possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1, col2, col3}, col1.InstructorRating + col2.InstructorRating + col3.InstructorRating})
					}
					continue
				}

				for _, col4 := range tempClasses[3].classes {
					if len(tempClasses) == 4 {
						// we stop here
						if isScheduleValid([]ClassEnhanced{col1, col2, col3, col4}) {
							possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1, col2, col3, col4}, col1.InstructorRating + col2.InstructorRating + col3.InstructorRating + col4.InstructorRating})
						}
						continue
					}

					for _, col5 := range tempClasses[4].classes {
						if len(tempClasses) == 5 {
							// we stop here
							if isScheduleValid([]ClassEnhanced{col1, col2, col3, col4, col5}) {
								possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1, col2, col3, col4, col5}, col1.InstructorRating + col2.InstructorRating + col3.InstructorRating + col4.InstructorRating + col5.InstructorRating})
							}
							continue
						}

						for _, col6 := range tempClasses[5].classes {
							if len(tempClasses) == 6 {
								// we stop here
								if isScheduleValid([]ClassEnhanced{col1, col2, col3, col4, col5, col6}) {
									possibleSchedules = append(possibleSchedules, Schedule{[]ClassEnhanced{col1, col2, col3, col4, col5, col6}, col1.InstructorRating + col2.InstructorRating + col3.InstructorRating + col4.InstructorRating + col5.InstructorRating + col6.InstructorRating})
								}
								continue
							}
						}
					}
				}
			}
		}
	}

	fmt.Println("Number of possible schedules: " + strconv.Itoa(len(possibleSchedules)))

	// Sorts all schedules by rating
	sort.Slice(possibleSchedules, func(i, j int) bool {
		return possibleSchedules[i].rating > possibleSchedules[j].rating
	})

	// Prints first ten best schedules - just printing it out for testing
	for i := 0; i < 10; i++ {
		fmt.Println("------------------")
		fmt.Println("Schedule #" + strconv.Itoa(i+1))
		fmt.Println("Rating: ")
		fmt.Println(possibleSchedules[i].rating)
		for _, class := range possibleSchedules[i].classes {
			fmt.Println(class.ClassID)
		}
	}

	return []ClassEnhanced{}
}

func isScheduleValid(classes []ClassEnhanced) bool {
	classTimeStart := 0
	classTimeEnd := 0

	iClassTimeStart := 0
	iClassTimeEnd := 0

	conflict := func(meetingTime MeetingTime, iMeetingTime MeetingTime) bool {

		// compare times
		classTimeStart = (meetingTime.StartTime.Hour * 100) + meetingTime.StartTime.Minute
		classTimeEnd = (meetingTime.EndTime.Hour * 100) + meetingTime.EndTime.Minute

		iClassTimeStart = (iMeetingTime.StartTime.Hour * 100) + iMeetingTime.StartTime.Minute
		iClassTimeEnd = (iMeetingTime.EndTime.Hour * 100) + iMeetingTime.EndTime.Minute

		if classTimeStart >= iClassTimeStart && classTimeStart <= iClassTimeEnd {
			// conflicts
			return true
		}

		if classTimeEnd >= iClassTimeStart && classTimeEnd <= iClassTimeEnd {
			// conflicts
			return true
		}

		return false
	}

	for _, class := range classes {
		for _, meetingTime := range class.MeetingTimes {

			for _, iClass := range classes {
				if class.ClassID == iClass.ClassID {
					continue
				}

				for _, iMeetingTime := range iClass.MeetingTimes {

					if meetingTime.Monday && iMeetingTime.Monday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Tuesday && iMeetingTime.Tuesday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Wednesday && iMeetingTime.Wednesday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Thursday && iMeetingTime.Thursday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Friday && iMeetingTime.Friday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Saturday && iMeetingTime.Saturday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}

					if meetingTime.Sunday && iMeetingTime.Sunday {
						if conflict(meetingTime, iMeetingTime) {
							return false
						}
					}
				}
			}
		}
	}

	return true
}

// Filters the courses we want and returns them
func filterCourses(school School, userScheduleConstraints UserScheduleConstraints) []Class {
	classes := []Class{}

	for _, class := range school.Classes {
		for _, courseName := range userScheduleConstraints.Courses {
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
			if JaroWinklerDistance(class.Instructor, professor.FirstName+" "+professor.MiddleName+" "+professor.LastName) > 0.90 {
				rating, err := strconv.ParseFloat(professor.OverallRating, 32)
				if err != nil {
					break
				}

				// USEFUL DEBUG INFO
				//fmt.Print("Instructor: " + class.Instructor)
				//fmt.Print(" RMF: " + professor.FirstName + " " + professor.MiddleName + " " + professor.LastName)
				//fmt.Print(" Accuracy: ")
				//fmt.Print(JaroWinklerDistance(class.Instructor, professor.FirstName+" "+professor.MiddleName+" "+professor.LastName))
				//fmt.Println()

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
		for _, preferredInstruction := range constraints.InstructionalMethods {
			if preferredInstruction == class.InstructionalMethod {
				fitsSchedule = true
			}
		}
		if !fitsSchedule {
			continue
		}
		fitsSchedule = false

		// Availability
		for _, preferredAvailability := range constraints.Availability {
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
					if available(meetingTime, constraints.MondayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// TUESDAY
				if meetingTime.Tuesday {
					if available(meetingTime, constraints.TuesdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// WEDNESDAY
				if meetingTime.Wednesday {
					if available(meetingTime, constraints.WednesdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// THURSDAY
				if meetingTime.Thursday {
					if available(meetingTime, constraints.ThursdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// FRIDAY
				if meetingTime.Friday {
					if available(meetingTime, constraints.FridayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// SATURDAY
				if meetingTime.Saturday {
					if available(meetingTime, constraints.SaturdayTime) {
						fitsSchedule = true
					} else {
						fitsSchedule = false
						break
					}
				}

				// SUNDAY
				if meetingTime.Sunday {
					if available(meetingTime, constraints.SundayTime) {
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

// JaroWinklerDistance Used this to get better matches between instructor names in schedule database and instructor names in rate my professor database
// Some of the names in the rate my professor database are not spelled the same as schedule database
// This function is an attempt to match names based on similarity.
// Ref: https://socketloop.com/tutorials/golang-how-to-find-out-similarity-between-two-strings-with-jaro-winkler-distance
func JaroWinklerDistance(s1, s2 string) float64 {

	s1Matches := make([]bool, len(s1)) // |s1|
	s2Matches := make([]bool, len(s2)) // |s2|

	var matchingCharacters = 0.0
	var transpositions = 0.0

	// sanity checks

	// return 0 if either one is empty string
	if len(s1) == 0 || len(s2) == 0 {
		return 0 // no similarity
	}

	// return 1 if both strings are empty
	if len(s1) == 0 && len(s2) == 0 {
		return 1 // exact match
	}

	if strings.EqualFold(s1, s2) { // case insensitive
		return 1 // exact match
	}

	// Two characters from s1 and s2 respectively,
	// are considered matching only if they are the same and not farther than
	// [ max(|s1|,|s2|) / 2 ] - 1
	matchDistance := len(s1)
	if len(s2) > matchDistance {
		matchDistance = len(s2)
	}
	matchDistance = matchDistance/2 - 1

	// Each character of s1 is compared with all its matching characters in s2
	for i := range s1 {
		low := i - matchDistance
		if low < 0 {
			low = 0
		}
		high := i + matchDistance + 1
		if high > len(s2) {
			high = len(s2)
		}
		for j := low; j < high; j++ {
			if s2Matches[j] {
				continue
			}
			if s1[i] != s2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matchingCharacters++
			break
		}
	}

	if matchingCharacters == 0 {
		return 0 // no similarity, exit early
	}

	// Count the transpositions.
	// The number of matching (but different sequence order) characters divided by 2 defines the number of transpositions
	k := 0
	for i := range s1 {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if s1[i] != s2[k] {
			transpositions++ // increase transpositions
		}
		k++
	}
	transpositions /= 2

	weight := (matchingCharacters/float64(len(s1)) + matchingCharacters/float64(len(s2)) + (matchingCharacters-transpositions)/matchingCharacters) / 3

	//  the length of common prefix at the start of the string up to a maximum of four characters
	l := 0

	// is a constant scaling factor for how much the score is adjusted upwards for having common prefixes.
	//The standard value for this constant in Winkler's work is {\displaystyle p=0.1}p=0.1
	p := 0.1

	// make it easier for s1[l] == s2[l] comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if weight > 0.7 {
		for (l < 4) && s1[l] == s2[l] {
			l++
		}

		weight = weight + float64(l)*p*(1-weight)
	}

	return weight
}
