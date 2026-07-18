--TEST--
P3 division and modulo edge semantics
--FILE--
<?php
echo (4 / 2 === 2.0 ? 'float' : 'bad');
echo ':';
echo (5 / 2 === 2.5 ? 'fraction' : 'bad');
echo ':';
echo ('12' / '3' === 4.0 ? 'string-float' : 'bad');
echo ':';
echo ('7' / '2' === 3.5 ? 'string-fraction' : 'bad');
echo "\n";
echo 5 % 2.9;
echo ':';
echo -5 % 2;
echo ':';
echo 5.9 % 2;
?>
--EXPECT--
float:fraction:string-float:string-fraction
1:-1:1
