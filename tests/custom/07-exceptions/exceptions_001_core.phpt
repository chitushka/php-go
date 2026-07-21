--TEST--
Exceptions 001: try/catch/finally-less (finally is PHP 5.5+, NOT tested here), multi-catch order, custom exceptions
--FILE--
<?php
// NOTE: `finally` blocks are a PHP 5.5+ feature and are intentionally NOT used
// in this PHP 5.3 test file.

class MyException extends Exception {}

function risky($n) {
    if ($n < 0) {
        throw new InvalidArgumentException("negative: $n");
    }
    if ($n === 0) {
        throw new MyException("zero not allowed");
    }
    return 100 / $n;
}

try {
    var_dump(risky(5));
} catch (Exception $e) {
    echo "should not happen\n";
}

try {
    risky(-1);
} catch (InvalidArgumentException $e) {
    var_dump($e->getMessage());
} catch (Exception $e) {
    echo "wrong catch\n";
}

try {
    risky(0);
} catch (MyException $e) {
    var_dump($e->getMessage());
} catch (Exception $e) {
    echo "wrong catch\n";
}

try {
    try {
        throw new Exception("inner");
    } catch (Exception $e) {
        throw new RuntimeException("outer");
    }
} catch (RuntimeException $e) {
    var_dump($e->getMessage());
}

function divide($a, $b) {
    if ($b === 0) {
        throw new Exception("division by zero");
    }
    return $a / $b;
}
try {
    divide(10, 0);
} catch (Exception $e) {
    var_dump(get_class($e));
    var_dump($e->getMessage());
    var_dump($e->getCode());
}
?>
--EXPECT--
int(20)
string(12) "negative: -1"
string(16) "zero not allowed"
string(5) "outer"
string(9) "Exception"
string(16) "division by zero"
int(0)
