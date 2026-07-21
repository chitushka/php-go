--TEST--
p1 float to string formatting matches PHP precision=14 (BUG-2)
--FILE--
<?php
echo 0.1 + 0.2, "\n";
echo 3.0, "\n";
echo 100000000000000.0, "\n";
echo 10000000000000.0, "\n";
echo 0.00001, "\n";
echo 0.0001, "\n";
var_dump(0.1 + 0.2);
var_dump(NAN);
var_dump(INF);
var_dump(-INF);
echo serialize(0.1 + 0.2), "\n";
?>
--EXPECT--
0.3
3
1.0E+14
10000000000000
1.0E-5
0.0001
float(0.3)
float(NAN)
float(INF)
float(-INF)
d:0.3;
