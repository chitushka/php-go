--TEST--
P1 assignment by reference for variables
--FILE--
<?php
$x = 1;
$alias =& $x;
$alias = 2;
echo $x;
echo ":";

$x = 3;
echo $alias;
echo ":";

$copy = $alias;
$copy = 9;
echo $x;
echo ":";
echo $alias;
echo ":";

$name = 'x';
$dyn =& $$name;
$dyn = 4;
echo $x;
echo ":";
echo $alias;
echo ":";

$leftName = 'left';
$rightName = 'x';
$$leftName =& $$rightName;
$left = 5;
echo $x;
echo ":";
echo $alias;
echo ":";
echo $dyn;
echo ":";

function local_ref() {
    $a = 'A';
    $b =& $a;
    $a = 'B';
    return $b;
}
echo local_ref();
?>
--EXPECT--
2:3:3:3:4:4:5:5:5:B
