--TEST--
P0 string escape sequences
--FILE--
<?php
echo "A\nB\t\x43\103";
echo "\n";
echo 'D\nE';
echo "\n";
echo 'it\'s \\ ok';
?>
--EXPECT--
A
B	CC
D\nE
it's \ ok
