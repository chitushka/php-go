--TEST--
P2 Exception accessors
--FILE--
<?php
try {
    throw new Exception('failed', 7);
} catch (Exception $e) {
    echo $e->getMessage();
    echo ':';
    echo $e->getCode();
    echo ':';
    echo basename($e->getFile());
    echo ':';
    echo $e->getLine();
    echo "\n";
    echo $e;
}
--EXPECTF--
failed:7:p2_exception_accessors.phpt:3
Exception: failed in %sp2_exception_accessors.phpt:3
