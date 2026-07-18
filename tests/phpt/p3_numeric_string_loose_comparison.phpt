--TEST--
P3 numeric strings in arithmetic and loose scalar comparisons
--FILE--
<?php
echo '12px' + 1;
echo ':';
echo ' 1.5e2rest' + 2;
echo ':';
echo 'abc' + 2;
echo "\n";
echo ('10' == 10 ? 'num' : 'no');
echo ':';
echo ('10abc' == 10 ? 'prefix' : 'no');
echo ':';
echo ('010' == '10' ? 'numeric-string' : 'no');
echo ':';
echo ('10abc' == '10' ? 'bad' : 'string');
echo "\n";
echo (0 == 'abc' ? 'zero' : 'no');
echo ':';
echo (false == '0' ? 'false' : 'no');
echo ':';
echo (null == '' ? 'null-empty' : 'no');
echo ':';
echo (null == '0' ? 'bad' : 'null-not-zero');
echo "\n";
echo ('10' < '2' ? 'bad' : 'numeric-order');
echo ':';
echo ('abc' < 'b' ? 'string-order' : 'bad');
?>
--EXPECT--
13:152:2
num:prefix:numeric-string:string
zero:false:null-empty:null-not-zero
numeric-order:string-order
