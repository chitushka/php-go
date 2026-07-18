--TEST--
P3 include_path resolution
--INI--
include_path=p3_include_path_lib
--FILE--
<?php
$value = include 'path_only.php';
echo $value;
echo "\n";
include 'shared.php';
echo "\n";
include 'p3_include_path_nested/loader.php';
echo "\n";
include './p3_include_path_same.php';
?>
--EXPECT--
from-path
path
local
dot
