package api

// SECURITY CONCERN, AUTHENTICATION ISADMIN FIELD CAN BE SET BY USER
import (
	"database/sql"
	"math/rand"

	//"encoding/json"
	// "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mabaums/ece461-web/backend/models"
	"github.com/mabaums/ece461-web/backend/packager"
	log "github.com/sirupsen/logrus"
)

type PackageRegEx struct {
	RegEx string `json:"regex"`
}

/*  HELPER FUNCTIONS */
func getDB(c *gin.Context) (*sql.DB, bool) {
	db_i, ok := c.Get("db")
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.Error{Code: 500, Message: "Server error"})
		return nil, false
	}
	db, ok := db_i.(*sql.DB)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.Error{Code: 500, Message: "Server error"})
		return nil, false
	}
	return db, true
}

func PackageCreate(c *gin.Context) {

	// Get Database
	db, ok := getDB(c)
	if !ok {
		return
	}

	if !authenticate(c) {
		return
	}
	/*v, _ := ioutil.ReadAll(c.Request.Body)
	log.Infof("Creating Package %v", string(v)) */
	// Process Request
	var data models.PackageData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	log.Infof("REQUEST -- PackageCreate -- data: %+v", data)

	// Verify that only one of Content and URl are set
	dataURLEmpty := len(data.URL) == 0
	dataContentEmpty := len(data.Content) == 0
	if !((dataURLEmpty && !dataContentEmpty) || (!dataURLEmpty && dataContentEmpty)) { // XOR
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"description": "There is missing field(s) in the PackageData" +
			"or it is formed improperly (e.g. Content and URL are both set)"})
		return
	}

	var metadata *models.PackageMetadata
	var err error
	var encoded string
	var ratings *models.PackageRating
	// Implement packages with content.
	if dataURLEmpty {
		c.AbortWithStatus(http.StatusNotImplemented)
		return
	} else {
		metadata, encoded, err = packager.GetPackageJson(data.URL)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest) // Handle server errors.
			return
		}
		log.Infof("Parsed package.json from %v, %+v", data.URL, metadata)
		log.Infof("Encoded zip file %v", encoded)
		data.Content = encoded

		ratings, err = packager.Rate(data.URL)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"description": "Error rating package"})
			return
		}
	}

	if !isGoodRating(ratings) {
		log.Infof("Pcakge has bad ratings. Ratings: %+v Pkg: %v", ratings, metadata.Name)
		c.AbortWithStatus(http.StatusFailedDependency)
		return
	}

	// Verify BOTH package name and version name are not the same. It's ok if package name is the same and versionis different.
	var exists bool
	err = db.QueryRow("SELECT * FROM PackageMetadata WHERE Name = ? AND Version = ?", metadata.Name, metadata.Version).Scan(&exists)
	if err != sql.ErrNoRows {
		// row does not exist, return an error response
		log.Infof("Found package with Name: %v, Version %v", metadata.Name, metadata.Version)
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"description": "Package exists already"})
		return
	}

	metadata.ID = generatePackageID(db)
	if metadata.ID == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error6"})
		return
	}

	log.Infof("Generated package id: %v", metadata.ID)
	// Insert PackageMetadata
	result, err := db.Exec("INSERT INTO PackageMetadata (Name, Version, PackageID) VALUES (?, ?, ?)", metadata.Name, metadata.Version, metadata.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"description": "Package exists already."})
		return
	}

	metadataID, err := result.LastInsertId()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error7"})
		return
	}

	log.Infof("MetadataID created %v", metadataID)
	// Insert PackageData

	var dataID int64
	result, err = db.Exec("INSERT INTO PackageData (Content, JSProgram) VALUES (?, ?)", data.Content, data.JSProgram)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error1"})
		return
	}
	dataID, err = result.LastInsertId()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error2"})
		return
	}

	// Insert Package
	result, err = db.Exec("INSERT INTO Package (metadata_id, data_id) VALUES (?, ?)", metadataID, dataID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error8"})
		return
	}

	packageID, err := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO PackageRating (package_id, BusFactor, Correctness, RampUp, ResponsiveMaintainer, LicenseScore, GoodPinningPractice)
	VALUES (?, ?, ?, ?, ?, ?, ?)`, packageID, ratings.BusFactor, ratings.Correctness, ratings.RampUp, ratings.ResponsiveMaintainer, ratings.LicenseScore, ratings.GoodPinningPractice)
	if err != nil {
		log.Errorf("Error inserting rating %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"description": "Internal server error: Could not insert package rating into database."})
		return
	}

	// Insert PackageHistoryEntry
	updatePackageHistory(c, int(metadataID), db, "CREATE")

	// Successful Response
	c.JSON(http.StatusCreated, gin.H{
		"metadata": metadata,
		"data":     data,
	})
}

// PackageUpdate - Update this content of the package.
// historyentry
func PackageUpdate(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	var pkg models.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Infof("REQUEST -- PackageUpdate -- Package: %+v", pkg)

	metadata := pkg.Metadata
	var existingPackage models.Package
	var package_data_id int
	var package_metadata_id int
	var package_id int

	err := db.QueryRow("SELECT p.id, p.data_id, pmd.id, pmd.Name, pmd.Version, pmd.PackageID "+
		"FROM Package p "+
		"JOIN PackageMetadata pmd ON p.metadata_id = pmd.id "+
		"WHERE pmd.Name = ? AND pmd.Version = ? AND pmd.PackageID = ?",
		metadata.Name, metadata.Version, metadata.ID).Scan(&package_id,
		&package_data_id, &package_metadata_id, &existingPackage.Metadata.Name, &existingPackage.Metadata.Version, &existingPackage.Metadata.ID,
	)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "Package does not exist"})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	haveURL := len(pkg.Data.URL) != 0
	var ratings *models.PackageRating

	if haveURL {
		metadata, encoded, err := packager.GetPackageJson(pkg.Data.URL)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest) // Handle server errors.
			return
		}
		log.Infof("Parsed package.json from %v, %+v", pkg.Data.URL, metadata)
		log.Infof("Encoded zip file %v", encoded)
		pkg.Data.Content = encoded

		ratings, err = packager.Rate(pkg.Data.URL)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"description": "Error rating package"})
			return
		}
	}

	if !isGoodRating(ratings) {
		log.Errorf("Updating pacakge %v, Ratings are no good: %+v", pkg.Metadata.Name, ratings)
		c.AbortWithStatus(http.StatusFailedDependency)
		return
	}

	// Update package data
	packageData := pkg.Data
	_, err = db.Exec("UPDATE PackageData pd SET Content = ?, JSProgram = ? WHERE pd.id = ? ",
		packageData.Content, packageData.JSProgram, package_data_id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	_, err = db.Exec(`UPDATE PackageRating pr SET BusFactor = ?, Correctness = ?, RampUp = ?, ResponsiveMaintainer = ?, LicenseScore = ?, GoodPinningPractice = ? WHERE pr.package_id = ?`,
		ratings.BusFactor, ratings.Correctness, ratings.RampUp, ratings.ResponsiveMaintainer, ratings.LicenseScore, ratings.GoodPinningPractice, package_id)
	if err != nil {
		log.Errorf("Error inserting rating %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"description": "Internal server error: Could not insert package rating into database."})
		return
	}

	// Insert PackageHistoryEntry
	updatePackageHistory(c, package_metadata_id, db, "UPDATE")

	c.JSON(http.StatusOK, gin.H{"description": "Version is updated"})
}

func generatePackageID(db *sql.DB) string {
	// Check Rating

	// PackageMetadata
	count := 0

	// Generate new ID if package ID already exists or if the id is not specified
	for {
		rand.Seed(time.Now().UnixNano())
		const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		b := make([]byte, 6)
		for i := range b {
			b[i] = chars[rand.Intn(len(chars))]
		}
		newID := string(b)

		err := db.QueryRow("SELECT COUNT(*) FROM PackageMetadata WHERE PackageID = ?", newID).Scan(&count)
		if err != nil {
			return ""
		}
		if count == 0 {
			return newID
		}
	}
}

func updatePackageHistory(c *gin.Context, package_metadata_id int, db *sql.DB, action string) {

	var User_temp models.User
	User_temp.Name = c.GetString("username")
	User_temp.IsAdmin = false
	packageHistoryEntry := models.PackageHistoryEntry{
		User:   User_temp,
		Date:   time.Now(),
		Action: action,
	}
	var user_table_id int
	err := db.QueryRow("SELECT User.id FROM User WHERE User.name= ?", User_temp.Name).Scan(&user_table_id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error9"})
		return
	}
	_, err = db.Exec("INSERT INTO PackageHistoryEntry (user_id, date, package_metadata_id, action) VALUES (?, ?, ?, ?)", user_table_id, packageHistoryEntry.Date, package_metadata_id, packageHistoryEntry.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

}

// PackageDelete - Delete this version of the package. given packageid
func PackageDelete(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	// Find and delete package history entries and then the package: note that this deletes the package version with the given ID and not necessarily all its versions
	packageID := strings.TrimLeft(c.Param("id"), "/")
	var metadataID int

	log.Infof("REQUEST -- Package Delete -- PackageID %v", packageID)

	err := db.QueryRow("SELECT id FROM PackageMetadata WHERE PackageID = ?", packageID).Scan(&metadataID)

	if err != nil {
		log.Errorf("SQL error, failed to get metadata id: %v", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "Package does not exist"})
		return
	}

	log.Infof("Retrieved metadataID %v", metadataID)

	_, err = db.Exec("DELETE FROM PackageHistoryEntry WHERE package_metadata_id = ?", metadataID)
	if err != nil {
		log.Errorf("SQL error: Failed to delete from history: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	var pkgID int
	err = db.QueryRow("SELECT id FROM Package WHERE metadata_id = ?", metadataID).Scan(&pkgID)

	_, err = db.Exec("DELETE FROM PackageRating WHERE package_id = ?", pkgID)
	if err != nil {
		log.Errorf("SQL error: Failed to delete from pacakge from package rating: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	_, err = db.Exec("DELETE  pmd, pd, p FROM Package p "+
		"LEFT JOIN PackageMetadata pmd ON p.metadata_id = pmd.id "+
		"LEFT JOIN PackageData pd ON p.data_id = pd.id "+
		"WHERE p.metadata_id = ?", metadataID)
	if err != nil {
		log.Errorf("SQL error: Failed to get package metadata and data: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"description": "Package is deleted"})
}

// PackageByNameDelete - Delete all versions of this package. return string of package name
func PackageByNameDelete(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	packageName := strings.TrimLeft(c.Param("name"), "/")
	log.Infof("REQUEST -- PackageByNameDelete -- %v", c.Param("name"))
	// Get the metadata IDs of all packages with the given name
	var metadataIDs []int
	rows, err := db.Query("SELECT id FROM PackageMetadata WHERE Name = ?", packageName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var metadataID int
		err = rows.Scan(&metadataID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		metadataIDs = append(metadataIDs, metadataID)
	}

	// Delete all history entries, package data, and package versions for each metadata ID
	for _, metadataID := range metadataIDs {
		_, err = db.Exec("DELETE FROM PackageHistoryEntry WHERE package_metadata_id = ?", metadataID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		var pkgID int
		err = db.QueryRow("SELECT id FROM Package WHERE metadata_id = ?", metadataID).Scan(&pkgID)

		_, err = db.Exec("DELETE FROM PackageRating WHERE package_id = ?", pkgID)
		if err != nil {
			log.Errorf("SQL error: Failed to delete from pacakge from package rating: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		_, err = db.Exec("DELETE  pmd, pd, p FROM Package p "+
			"LEFT JOIN PackageMetadata pmd ON p.metadata_id = pmd.id "+
			"LEFT JOIN PackageData pd ON p.data_id = pd.id "+
			"WHERE p.metadata_id = ?", metadataID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"description": "Package is deleted"})
}

// PackageRetrieve - Interact with the package with this ID
// historyentry
func PackageRetrieve(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	if !authenticate(c) {
		return
	}

	log.Infof("REQUEST -- PackageRetrieve -- %v", c.Param("id"))

	packageID := strings.TrimLeft(c.Param("id"), "/")

	var packageContent, packageURL, packageJSProgram sql.NullString
	var packageName, packageVersion, packageContentS, packageURLS, packageJSProgramS string
	var package_data_id int

	err := db.QueryRow("SELECT p.data_id, m.Name, m.Version "+
		"FROM Package p "+
		"JOIN PackageMetadata m ON p.metadata_id = m.id "+
		"WHERE m.PackageID = ?", packageID).Scan(&package_data_id, &packageName, &packageVersion)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "Package does not exist."})
		return
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	fmt.Print(package_data_id)

	err = db.QueryRow("SELECT Content, URL, JSProgram FROM PackageData WHERE id = ?", package_data_id).Scan(&packageContent, &packageURL, &packageJSProgram)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "Package does not exist."})
			return
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error1": err})
			return
		}
	}
	if packageContent.Valid {
		packageContentS = packageContent.String
	} else {
		packageContentS = ""
	}
	if packageURL.Valid {
		packageURLS = packageURL.String
	} else {
		packageURLS = ""
	}
	if packageJSProgram.Valid {
		packageJSProgramS = packageJSProgram.String
	} else {
		packageJSProgramS = ""
	}
	// do something with retrieved data

	metadata := models.PackageMetadata{
		ID:      packageID,
		Name:    packageName,
		Version: packageVersion,
	}
	data := models.PackageData{
		Content:   packageContentS,
		URL:       packageURLS,
		JSProgram: packageJSProgramS,
	}

	// updatePackageHistory(c, 0) need to update package history, find way to obtain from database

	c.JSON(http.StatusOK, gin.H{
		"metadata": metadata,
		"data":     data,
	})
}

// RegistryReset - Reset the registry
func RegistryReset(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	log.Infof("REQUEST -- RegistryReset")

	// verify admin status

	if !c.GetBool("admin") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"description": "You do not have permission to reset the registry"})
		return
	}

	// Delete all data from Package table
	_, err := db.Exec("DELETE FROM PackageRating")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Delete all data from Package table
	_, err = db.Exec("DELETE FROM Package")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Delete all data from PackageData table
	_, err = db.Exec("DELETE FROM PackageData")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Delete all data from PackageHistoryEntry table
	_, err = db.Exec("DELETE FROM PackageHistoryEntry")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Delete all data from PackageMetadata table
	_, err = db.Exec("DELETE FROM PackageMetadata")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"description": "Registry is reset"})
}

// PacakgeByNameGet -
func PackageByNameGet(c *gin.Context) {
	// Return the history of this package (all versions).
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}
	// Get name from query parameter
	name := strings.TrimLeft(c.Param("name"), "/")

	// Query for all packages with matching name
	rows, err := db.Query("SELECT pmd.Name, pmd.Version, pmd.PackageID, phe.user_id, phe.date, phe.action "+
		"FROM Package p "+
		"JOIN PackageMetadata pmd ON p.metadata_id = pmd.id "+
		"JOIN PackageHistoryEntry phe ON p.metadata_id = phe.package_metadata_id "+
		"WHERE pmd.Name = ?", name)

	fmt.Print(rows)
	if !rows.Next() || err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "No such package."})
		return
	}
	defer rows.Close()

	// Store results in a slice of PackageHistoryEntry structs
	packageHistoryEntries := make([]models.PackageHistoryEntry, 0)
	for rows.Next() {
		var packageHistoryEntry models.PackageHistoryEntry
		var packageMetadata models.PackageMetadata
		var user_id int
		err := rows.Scan(&packageMetadata.Name, &packageMetadata.Version, &packageMetadata.ID,
			&user_id, &packageHistoryEntry.Date, &packageHistoryEntry.Action)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		err = db.QueryRow("SELECT * FROM User WHERE id = ?", user_id).Scan(&packageHistoryEntry.User)

		packageHistoryEntry.PackageMetadata = packageMetadata
		packageHistoryEntries = append(packageHistoryEntries, packageHistoryEntry)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, packageHistoryEntries)
}

func PackagesList(c *gin.Context) {
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	// Parse query parameters
	limitStr := c.Query("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10 // default limit
	}

	offsetStr := c.Query("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0 // default offset
	}

	// Parse request body
	var packageQueries []models.PackageQuery
	err = c.BindJSON(&packageQueries)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Infof("REQUEST -- PackagesList -- Queries: %+v, Offset: %v", packageQueries, c.Query("offset"))

	queryStr := "SELECT * FROM PackageMetadata"
	clausesStr := ""
	params := []interface{}{}
	for _, query := range packageQueries {
		clause := ""
		if query.Name != "" && query.Name != "*" {
			clause += "NAME = ?"
			params = append(params, query.Name)
		}
		if query.Name == "*" {
			clause += "NAME LIKE '%'"
		}
		if query.Version != "" {
			if clause != "" {
				clause += " AND "
			}
			vClause, vParams, _ := convertToBasicComparisons(query.Version)
			clause += vClause
			params = append(params, vParams...)
		}
		if clausesStr != "" {
			clausesStr += " OR (" + clause + ")"
		} else {
			clausesStr += "(" + clause + ")"
		}
	}
	if clausesStr != "" {
		queryStr += " WHERE " + clausesStr
	}
	queryStr += fmt.Sprintf(" LIMIT ? OFFSET ?")
	params = append(params, limit)
	params = append(params, offset)
	//queryStr := fmt.Sprintf("SELECT * FROM PackageMetadata WHERE Name = %s AND Version REGEXP '^[^.]+\\.[^.]+\\.[^.]+$' AND  %s LIMIT %d OFFSET %d;", strings.Join(nameConditions, ","), strings.Join(rangeConditions, " AND "), limit, offset)
	// fmt.Print(queryStr)
	log.Infof("Querying DB %v", queryStr)

	rows, err := db.Query(queryStr, params...)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var packages []models.PackageMetadata
	for rows.Next() {
		var p models.PackageMetadata
		if err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.ID); err != nil {
			panic(err)
		}
		packages = append(packages, p)
	}

	log.Infof("Packages retreived from db %+v", packages)

	// TO FIX return offset header.
	if len(packages) == limit {
		log.Infof("Returned 10 packages, could be more.")
		c.Header("offset", strconv.Itoa(offset+limit))
	}

	c.JSON(http.StatusOK, packages)
}

// UTILITY FOR PACKAGESLIST
func convertToBasicComparisons(v string) (string, []interface{}, error) {
	params := []interface{}{}
	clauseStr := ""
	if strings.HasPrefix(v, "^") {
		vParsed := semver.MustParse(v[1:])
		clauseStr += "Version >= ? AND Version < ?"
		params = append(params, fmt.Sprintf("%d.%d", vParsed.Major(), vParsed.Minor()))
		params = append(params, fmt.Sprintf("%d.%d", vParsed.Major()+1, 0))
	} else if strings.HasPrefix(v, "~") {
		vParsed := semver.MustParse(v[1:])
		clauseStr += "Version >= ? AND Version < ?"
		params = append(params, fmt.Sprintf("%d.%d", vParsed.Major(), vParsed.Minor()))
		params = append(params, fmt.Sprintf("%d.%d", vParsed.Major(), vParsed.Minor()+1))
	} else if strings.Contains(v, "-") {
		bounds := strings.Split(v, "-")
		if len(bounds) != 2 {
			return "", nil, fmt.Errorf("invalid bounded range: %s", v)
		}
		bounds[0] = strings.TrimSuffix(bounds[0], ".")
		params = append(params, bounds[0], bounds[1])
		clauseStr += "Version >= ? AND Version < ?"
	} else {
		params = append(params, v)
		clauseStr += "Version = ?"
	}
	return clauseStr, params, nil
}

// PackageByRegExGet - Get any packages fitting the regular expression.
func PackageByRegExGet(c *gin.Context) {
	// Search for packages that match the regular expression
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	// Parse the request body as a PackageRegEx object
	var query PackageRegEx
	err := c.ShouldBindJSON(&query)
	log.Infof("REQUEST -- PackageByRegExGet -- %+v", query)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var packages []models.PackageMetadata
	fmt.Print(query)
	rows, err := db.Query("SELECT version, name, id FROM packagemetadata WHERE name REGEXP ?", query.RegEx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var pkg models.PackageMetadata
		err := rows.Scan(&pkg.Version, &pkg.Name, &pkg.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		packages = append(packages, pkg)
	}
	if err := rows.Err(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Return the packages as a JSON array
	if len(packages) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"description": "No package found under this regex."})
		return
	}
	c.JSON(http.StatusOK, packages)

}

// PackageRate -
// historyentry
func PackageRate(c *gin.Context) {
	// TODO INSERT INTO HISTORY THAT WE HAVE RATED
	db, ok := getDB(c)
	if !ok {
		return
	}

	// Authentication
	if !authenticate(c) {
		return
	}

	log.Infof("REQUEST -- PackageRate -- %v", c.Param("id"))
	pkgID := c.Param("id")
	if pkgID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`SELECT BusFactor, Correctness, RampUp, ResponsiveMaintainer, LicenseScore, GoodPinningPractice FROM PackageRating pr
	WHERE pr.package_id = (SELECT id FROM Package WHERE metadata_id = (SELECT id FROM PackageMetadata WHERE PackageID = ?))`, pkgID)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pkgRating models.PackageRating
		err := rows.Scan(&pkgRating.BusFactor, &pkgRating.Correctness, &pkgRating.RampUp, &pkgRating.ResponsiveMaintainer, &pkgRating.LicenseScore, &pkgRating.GoodPinningPractice)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		c.JSON(http.StatusOK, pkgRating)
		return
	}
	if err := rows.Err(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.AbortWithStatus(http.StatusNotFound)
}

func isGoodRating(pkgRating *models.PackageRating) bool {
	return pkgRating.LicenseScore >= 0.5
}
