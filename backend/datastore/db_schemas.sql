CREATE DATABASE ECE461;
USE ECE461;

CREATE TABLE PackageMetadata (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  Name VARCHAR(255),
  Version VARCHAR(255),
  PackageID VARCHAR(255),
  PRIMARY KEY (id)
);

CREATE TABLE PackageData (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  Content LONGTEXT,
  URL VARCHAR(255),
  JSProgram LONGTEXT,
  PRIMARY KEY (id)
);

CREATE TABLE Package (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  metadata_id INT UNSIGNED,
  data_id INT UNSIGNED,
  PRIMARY KEY (id),
  FOREIGN KEY (metadata_id) REFERENCES PackageMetadata(id) ON DELETE CASCADE,
  FOREIGN KEY (data_id) REFERENCES PackageData(id) ON DELETE CASCADE
);


CREATE TABLE User (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255),
  isAdmin BOOLEAN,
  PRIMARY KEY (id)
);

CREATE TABLE UserAuthenticationInfo (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id INT UNSIGNED,
  password VARCHAR(255),
  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES User(id)
);

CREATE TABLE PackageRating (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  package_id INT UNSIGNED,
  BusFactor DOUBLE,
  Correctness DOUBLE,
  RampUp DOUBLE,
  ResponsiveMaintainer DOUBLE,
  LicenseScore DOUBLE,
  GoodPinningPractice DOUBLE,
  NetScore DOUBLE,
  PullRequest DOUBLE,
  PRIMARY KEY (id),
  FOREIGN KEY (package_id) REFERENCES Package(id)
);

CREATE TABLE PackageHistoryEntry (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id INT UNSIGNED,
  date DATETIME,
  package_metadata_id INT UNSIGNED,
  action VARCHAR(255),
  PRIMARY KEY (id),
  FOREIGN KEY (user_id) REFERENCES User(id),
  FOREIGN KEY (package_metadata_id) REFERENCES PackageMetadata(id)
);

-- Creating Admin foo, bar for testing
INSERT INTO User (name, isAdmin) VALUES ("foo", True);
INSERT INTO UserAuthenticationInfo (user_id, password) VALUES (1, "bar");