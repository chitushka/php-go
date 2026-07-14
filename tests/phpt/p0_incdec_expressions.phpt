--TEST--
P0 increment and decrement expression values
--FILE--
<?php
$a = 1;
echo $a++;
echo ':';
echo $a;
echo ':';
echo ++$a;
echo ':';
echo $a--;
echo ':';
echo $a;
echo ':';
echo --$a;
echo "\n";

$s = 'a';
echo $s++;
echo ':';
echo $s;
echo ':';
$z = 'z';
echo ++$z;
echo ':';
$n = '9';
echo $n++;
echo ':';
echo $n;
echo ':';
$empty = '';
echo ++$empty;
echo ':';
$word = 'b';
echo $word--;
echo ':';
echo $word;
echo ':';
$num = '5';
echo --$num;
?>
--EXPECT--
1:2:3:3:2:1
a:b:aa:9:10:1:b:b:4
