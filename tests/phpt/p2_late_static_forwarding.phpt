--TEST--
P2 late static binding forwarding calls
--FILE--
<?php
class ForwardBase {
    public static $name = 'base';

    public static function who() {
        return get_called_class() . ':' . static::$name;
    }

    public static function __callStatic($name, $args) {
        return get_called_class() . ':' . $name . ':' . static::$name;
    }
}

class ForwardChild extends ForwardBase {
    public static $name = 'child';

    public static function viaParent() {
        return parent::who();
    }

    public static function viaSelf() {
        return self::who();
    }

    public static function viaBase() {
        return ForwardBase::who();
    }

    public static function missingParent() {
        return parent::missing();
    }

    public static function missingSelf() {
        return self::missing();
    }

    public static function missingBase() {
        return ForwardBase::missing();
    }
}

echo ForwardChild::viaParent();
echo "\n";
echo ForwardChild::viaSelf();
echo "\n";
echo ForwardChild::viaBase();
echo "\n";
echo ForwardChild::missingParent();
echo "\n";
echo ForwardChild::missingSelf();
echo "\n";
echo ForwardChild::missingBase();
?>
--EXPECT--
ForwardChild:child
ForwardChild:child
ForwardBase:base
ForwardChild:missing:child
ForwardChild:missing:child
ForwardBase:missing:base
