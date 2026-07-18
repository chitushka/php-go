--TEST--
P2 destructor does not run when __clone throws
--FILE--
<?php
class BadClone {
    function __clone() {
        throw new Exception('boom');
    }

    function __destruct() {
        echo ':destruct';
    }
}

$obj = new BadClone();
try {
    clone $obj;
} catch (Exception $e) {
    echo 'caught';
}
echo ':done';
?>
--EXPECT--
caught:done:destruct
