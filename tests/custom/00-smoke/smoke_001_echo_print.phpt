--TEST--
Smoke 001: echo, print, print_r, var_dump basic output
--FILE--
<?php
echo "hello\n";
echo "a", "b", "c", "\n";
print "world\n";
$x = print "chained\n";
var_dump($x);
print_r(array(1, "two", 3.0));
echo "\n";
var_dump(true, false, null, 1, 1.5, "s");
?>
--EXPECT--
hello
abc
world
chained
int(1)
Array
(
    [0] => 1
    [1] => two
    [2] => 3
)

bool(true)
bool(false)
NULL
int(1)
float(1.5)
string(1) "s"
