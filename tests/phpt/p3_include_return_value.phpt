--TEST--
P3 include and include_once expression return values
--FILE--
<?php
$seen = 0;
$a = include 'p3_include_return_value_lib.php';
echo $a;
echo ':';
echo $seen;
echo "\n";
$b = include 'p3_include_no_return_lib.php';
echo ($b === 1 ? 'one' : 'bad');
echo "\n";
$c = include 'p3_include_return_null_lib.php';
echo ($c === null ? 'null' : 'bad');
echo "\n";
$d = include_once 'p3_include_once_return_lib.php';
echo $d;
echo "\n";
$e = include_once './p3_include_once_return_lib.php';
echo ($e === true ? 'true' : 'bad');
echo "\n";
echo $seen;
?>
--EXPECT--
ret:1
one
null
once
true
3
