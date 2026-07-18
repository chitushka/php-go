--TEST--
P2 method_exists and property_exists autoload string targets
--FILE--
<?php
function __autoload($class) {
    include 'p2_method_property_exists_autoload_lib.php';
}

echo method_exists('ExistsAutoloaded', 'run') ? 'method' : 'no';
echo "\n";
echo method_exists('ExistsAutoloadContract', 'run') ? 'iface' : 'no';
echo "\n";
echo property_exists('ExistsAutoloaded', 'name') ? 'prop' : 'no';
echo "\n";
echo property_exists('ExistsAutoloaded', 'secret') ? 'private' : 'no';
?>
--EXPECT--
method
iface
prop
private
