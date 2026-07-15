--TEST--
P2 Exception getTrace array
--FILE--
<?php
class Runner {
    public function wrap() {
        fail();
    }
}

function fail() {
    throw new Exception('outer');
}

try {
    $runner = new Runner();
    $runner->wrap();
} catch (Exception $e) {
    $trace = $e->getTrace();
    echo $trace[0]['function'];
    echo "\n";
    echo $trace[1]['class'];
    echo ':';
    echo $trace[1]['type'];
    echo ':';
    echo $trace[1]['function'];
}
--EXPECT--
fail
Runner::::wrap
