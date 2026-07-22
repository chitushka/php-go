--TEST--
Exception chaining 001 (NEW in PHP 5.3): previous exception parameter and getPrevious()
--FILE--
<?php
class DatabaseException extends Exception {}
class ServiceException extends Exception {}

function connectToDatabase() {
    throw new DatabaseException("connection refused", 500);
}

function handleRequest() {
    try {
        connectToDatabase();
    } catch (DatabaseException $e) {
        // NEW in PHP 5.3: Exception::__construct($message, $code, $previous)
        throw new ServiceException("service unavailable", 0, $e);
    }
}

try {
    handleRequest();
} catch (ServiceException $e) {
    var_dump($e->getMessage());
    $prev = $e->getPrevious();
    var_dump($prev instanceof DatabaseException);
    var_dump($prev->getMessage());
    var_dump($prev->getCode());
    var_dump($prev->getPrevious());
}

// Chain of 3
try {
    try {
        try {
            throw new Exception("level-1");
        } catch (Exception $e1) {
            throw new Exception("level-2", 0, $e1);
        }
    } catch (Exception $e2) {
        throw new Exception("level-3", 0, $e2);
    }
} catch (Exception $top) {
    $messages = array();
    $current = $top;
    while ($current !== null) {
        $messages[] = $current->getMessage();
        $current = $current->getPrevious();
    }
    var_dump($messages);
}

// No previous exception given -> getPrevious() returns null
$e = new Exception("standalone");
var_dump($e->getPrevious());
--EXPECT--
string(19) "service unavailable"
bool(true)
string(18) "connection refused"
int(500)
NULL
array(3) {
  [0]=>
  string(7) "level-3"
  [1]=>
  string(7) "level-2"
  [2]=>
  string(7) "level-1"
}
NULL
