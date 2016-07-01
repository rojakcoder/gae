# Change Log

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
