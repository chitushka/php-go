--TEST--
p0 logical assignment precedence
--FILE--
<?php
$b = true or false;
echo $b ? '1' : '0';
echo ':';
$b = false or true;
echo $b ? '1' : '0';
echo ':';
$b = true || false;
echo $b ? '1' : '0';
echo ':';
$b = false || true;
echo $b ? '1' : '0';
echo ':';
$b = true and false;
echo $b ? '1' : '0';
echo ':';
$b = true && false;
echo $b ? '1' : '0';
--EXPECT--
1:0:1:1:1:0
