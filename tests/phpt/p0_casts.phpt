--TEST--
p0 casts
--FILE--
<?php
echo (int) '12px';
echo ':';
echo (integer) true;
echo ':';
echo (float) '1.5x';
echo ':';
echo (string) 7;
echo ':';
echo (bool) '';
echo ':';
echo (boolean) 'x';
echo ':';
$a = (array) 'v';
echo count($a);
echo ':';
echo $a[0];
echo ':';
$empty = (array) null;
echo count($empty);
echo ':';
$o = (object) array('a' => 9);
echo $o->a;
echo ':';
$s = (object) 3;
echo $s->scalar;
echo ':';
echo (unset) 'x';
echo 'done';
--EXPECT--
12:1:1.5:7::1:1:v:0:9:3:done
