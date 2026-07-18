--TEST--
P3 foreach non-array warning is non-halting by default
--FILE--
<?php
foreach (1 as $value) {
    echo 'bad';
}
echo 'after';
?>
--EXPECT--
after
