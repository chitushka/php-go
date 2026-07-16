--TEST--
P2 constructor visibility
--FILE--
<?php
class PrivateFactory {
    private function __construct() {
        $this->value = 'private';
    }

    public static function create() {
        return new self();
    }

    public function value() {
        return $this->value;
    }
}

class ProtectedBaseFactory {
    protected function __construct() {
        $this->value = 'protected';
    }
}

class ProtectedChildFactory extends ProtectedBaseFactory {
    public static function create() {
        return new self();
    }

    public function value() {
        return $this->value;
    }
}

class PrivateOldStyleFactory {
    private function PrivateOldStyleFactory() {
        $this->value = 'old-private';
    }

    public static function create() {
        return new self();
    }

    public function value() {
        return $this->value;
    }
}

echo PrivateFactory::create()->value();
echo "\n";
echo ProtectedChildFactory::create()->value();
echo "\n";
echo PrivateOldStyleFactory::create()->value();
?>
--EXPECT--
private
protected
old-private
