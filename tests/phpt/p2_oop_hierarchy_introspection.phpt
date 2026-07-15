--TEST--
P2 OOP hierarchy introspection functions
--FILE--
<?php
function __autoload($class) {
    include 'p2_oop_hierarchy_introspection_lib.php';
}

$obj = new HierChild();
echo get_parent_class($obj);
echo ':';
echo $obj->parentName();
echo "\n";

echo is_subclass_of($obj, 'HierBase') ? 'sub' : 'no';
echo ':';
echo is_subclass_of('HierChild', 'ParentContract') ? 'iface' : 'no';
echo "\n";

echo is_a($obj, 'HierChild') ? 'same' : 'no';
echo ':';
echo is_a('HierChild', 'HierBase') ? 'string' : 'no';
echo ':';
echo is_a('HierChild', 'HierBase', true) ? 'allowed' : 'no';
--EXPECT--
HierBase:HierBase
sub:iface
same:no:allowed
