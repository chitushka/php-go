--TEST--
P2 OOP hierarchy array introspection functions
--FILE--
<?php
function __autoload($class) {
    include 'p2_oop_hierarchy_introspection_lib.php';
}

echo class_parents('HierChild', false) ? 'loaded' : 'missing';
echo "\n";

$parents = class_parents('hierchild');
echo $parents['HierBase'];
echo "\n";

$interfaces = class_implements('HierChild');
echo $interfaces['ChildContract'];
echo ':';
echo $interfaces['ParentContract'];
echo "\n";

$ifaceParents = class_implements('ChildContract');
echo $ifaceParents['ParentContract'];
--EXPECT--
missing
HierBase
ChildContract:ParentContract
ParentContract
