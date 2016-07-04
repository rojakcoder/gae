# Change Log

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
