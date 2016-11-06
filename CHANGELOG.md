# Change Log

## [0.10.0] - 2016-11-06

### Added
- Added the Session struct for managing session.

## [0.9.0] - 2016-08-24

### Added
- Added a String method to DateTime to return the time in the RFC3339 format.

### Changed
- Changed NewDateTime function to accept a string in the format of RFC3339 to
create a DateTime instance; previously it was expecting a JSON string - the
difference being that the latter requires the time string to be in quotes.
- Updated the test to keep 100% code coverage.

## [0.8.0] - 2016-08-19

### Changed
- Added a few helper methods for interacting with Memcache.

## [0.7.0] - 2016-07-20

### Added
- Added a new convenient method for creating DateTime instance for the current
time.

### Changed
- Changed the sequence of header writing for WriteJSONColl to avoid the
warning.

## [0.6.1] - 2016-07-19

### Changed
- Changed the sequence of writing response header to avoid the warning of
multiple write header calls.

## [0.6.0] - 2016-07-17

### Added
- Added a new function for easily creating new DateTime instances.

### Changed
- Fixed a bug with date unmarshalling that does not handle empty quotes.
- Achieved 100% coverage on DateTime and methods.

## [0.5.2] - 2016-07-07

### Changed
- Achieved 100% code coverage.

## [0.5.1] - 2016-07-06

### Added
- Added more tests to validate the library.

### Changed
- Fixed a bug that causes Save to fail.

## [0.5.0] - 2016-07-04

### Added
- Added a new generic error struct.

## [0.4.0] - 2016-07-04

### Added
- Added a new error for type incompatability.
- Added Update as a method for the Model interface.
- Added getter and setter for the key field for Model.

### Changed
- Changed the ID method from the Model interface into a function - the
rationale is that the code is identical for all the models.

## [0.3.0] - 2016-07-03

### Removed
- Removed the method SetKey from the Model interface.

## [0.2.0] - 2016-07-02

### Added
- Added a new error value for missing ID as path parameter.
- Added a ID method to the Model interface for converting Key into string.
- Added a WriteJSONColl function to send the collection of model to the
response together with a pagination cursor.

### Changed
- Changed the IsValid method into a function.

### Removed
- Removed the ReadJSON function as it is redundant.

## [0.1.0] - 2016-07-01

### Added
- Added header for cursor pagination.

### Changed
- Improved the error string for different error structs.

## [0.0.1] - 2016-06-30

Initial commit
