--TEST--
P2 multiple catch blocks
--FILE--
<?php
class FirstError extends Exception {}
class SecondError extends Exception {}

try {
    throw new SecondError('second');
} catch (FirstError $e) {
    echo 'first';
} catch (SecondError $e) {
    echo $e->getMessage();
} catch (Exception $e) {
    echo 'base';
}
?>
--EXPECT--
second
