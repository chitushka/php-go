--TEST--
P0 PHP 5.3 string bitwise operators
--FILE--
<?php
$and = 'ABCD' & 'ab';
echo $and;
echo ':';
echo strlen($and);
echo "\n";
$or = 'AB' | 'abc';
echo $or;
echo ':';
echo strlen($or);
echo "\n";
$xor = 'AZ!' ^ '  ';
echo $xor;
echo ':';
echo strlen($xor);
echo "\n";
echo ~"\x9e\x85";
echo "\n";
echo '12' | 1;
?>
--EXPECT--
AB:2
abc:3
az:2
az
13
