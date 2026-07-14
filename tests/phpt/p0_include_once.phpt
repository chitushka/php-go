--TEST--
p0 include_once and require_once
--FILE--
<?php
$count = 0;
include_once 'p0_include_once_lib.php';
include_once './p0_include_once_lib.php';
require_once 'p0_include_once_lib.php';
echo ':';
echo $count;
?>
--EXPECT--
I:1
