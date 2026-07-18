--TEST--
P2 destructor does not run when constructor throws
--FILE--
<?php
class BadConstructor {
    function __construct() {
        throw new Exception('boom');
    }

    function __destruct() {
        echo ':destruct';
    }
}

try {
    new BadConstructor();
} catch (Exception $e) {
    echo 'caught';
}
echo ':done';
?>
--EXPECT--
caught:done
