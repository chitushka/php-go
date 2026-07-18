--TEST--
P2 destructor does not run when __wakeup throws
--FILE--
<?php
class BadWake {
    public function __wakeup() {
        throw new Exception('boom');
    }

    public function __destruct() {
        echo ':destruct';
    }
}

try {
    unserialize('O:7:"BadWake":0:{}');
} catch (Exception $e) {
    echo 'caught';
}
echo ':done';
?>
--EXPECT--
caught:done
