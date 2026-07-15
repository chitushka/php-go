--TEST--
P2 Exception previous and trace string
--FILE--
<?php
function wrap() {
    fail();
}

function fail() {
    throw new Exception('outer', 2, new Exception('inner', 1));
}

try {
    wrap();
} catch (Exception $e) {
    $previous = $e->getPrevious();
    echo $previous->getMessage();
    echo ':';
    echo $previous->getCode();
    echo "\n";
    echo $e->getTraceAsString();
}
--EXPECT--
inner:1
#0 fail()
#1 wrap()
