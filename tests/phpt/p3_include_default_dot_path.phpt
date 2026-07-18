--TEST--
P3 default dot include_path precedes included file directory
--FILE--
<?php
include 'p3_include_default_dot_path_nested/plain.php';
echo "\n";
include 'p3_include_default_dot_path_nested/explicit_dot.php';
?>
--EXPECT--
dot
nested
