--TEST--
Late Static Binding 001 (NEW in PHP 5.3): static::, get_called_class(), forward_static_call
--FILE--
<?php
class ParentModel {
    public static function create() {
        return new static();
    }
    public static function who() {
        return static::class_name();
    }
    public static function class_name() {
        return __CLASS__;
    }
    public static function getCalled() {
        return get_called_class();
    }
    public static function selfVsStatic() {
        return self::class_name() . ' / ' . static::class_name();
    }
}

class ChildModel extends ParentModel {
    public static function class_name() {
        return __CLASS__;
    }
}

var_dump(get_class(ParentModel::create()));
var_dump(get_class(ChildModel::create()));

var_dump(ParentModel::who());
var_dump(ChildModel::who());

var_dump(ParentModel::getCalled());
var_dump(ChildModel::getCalled());

var_dump(ChildModel::selfVsStatic());

class GrandChildModel extends ChildModel {}
var_dump(GrandChildModel::who());
var_dump(get_class(GrandChildModel::create()));

class Base {
    public static function test() {
        return forward_static_call(array(get_called_class(), 'implementation'));
    }
    public static function implementation() {
        return 'Base::implementation';
    }
}
class Derived extends Base {
    public static function implementation() {
        return 'Derived::implementation';
    }
}
var_dump(Base::test());
var_dump(Derived::test());
?>
--EXPECT--
string(11) "ParentModel"
string(10) "ChildModel"
string(11) "ParentModel"
string(10) "ChildModel"
string(11) "ParentModel"
string(10) "ChildModel"
string(24) "ParentModel / ChildModel"
string(10) "ChildModel"
string(15) "GrandChildModel"
string(20) "Base::implementation"
string(23) "Derived::implementation"
